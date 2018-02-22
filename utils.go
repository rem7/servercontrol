package servercontrol

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const ISO_8601 = "2006-01-02T15:04:05.999Z"

var (
	sess    *session.Session
	ASG     *autoscaling.AutoScaling
	EC2     *ec2.EC2
	ec2Meta *ec2metadata.EC2Metadata

	gRegion     string
	gInstanceId string
	gUserData   string

	lcRegex = regexp.MustCompile(`(.*)-(\d+)`)
)

type Instance struct {
	InstanceID     string `json:"instance_id"`
	ImageID        string `json:"image_id"`
	State          string `json:"state"`
	InstanceType   string `json:"instance_type"`
	LaunchDatetime string `json:"launch_datetime"`
	PrivateIP      string `json:"private_ip"`
	PublicIP       string `json:"public_ip"`
	GitCommitHash  string `json:"git_commit_hash"`
	StartTime      string `json:"start_time"`
	Hostname       string `json:"hostname"`
}

type Group struct {
	Name                string       `json:"name"`
	LaunchConfiguration LaunchConfig `json:"launch_configuration"`
	instances           []*autoscaling.Instance
}

func (g *Group) Instances() []*autoscaling.Instance {
	return g.instances
}

type LaunchConfig struct {
	Name     string `json:"name"`
	ImageID  string `json:"image_id"`
	UserData string `json:"user_data"`
}

type ServiceData struct {
	MasterGitHash  string     `json:"master_git_hash"`
	InstanceID     string     `json:"instance_id"`
	InstanceList   []Instance `json:"instance_list"`
	AutoScaleGroup Group      `json:"auto_scale_group"`
}

type defaultProps struct {
	Hash   string `json:"hash"`
	Secret string `json:"secret"`
}

func parseDefaultProps(req *http.Request, res http.ResponseWriter) (defaultProps, error) {

	defer req.Body.Close()

	props := defaultProps{}
	err := json.NewDecoder(req.Body).Decode(&props)
	switch {
	case err == io.EOF:
		res.WriteHeader(http.StatusBadRequest)
		return props, err
	case err != nil:
		res.WriteHeader(http.StatusInternalServerError)
		return props, err
	}

	return props, nil

}

func parseBody(body io.ReadCloser, i interface{}) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(&i)
}

func apiRequest(url, method string, body io.Reader) (*http.Response, error) {

	client := &http.Client{Timeout: time.Second * 30}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Sc-Secret", gConfig.Secret)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func getServiceForInstance(instance Instance, service string) string {

	url := fmt.Sprintf("%s://%s:%d%s/%s", gConfig.Proto,
		instance.PrivateIP, gConfig.ServicePort, gConfig.Prefix, service)
	return url
}

func getInstances(instanceIds []*string) ([]Instance, error) {

	ec2params := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	}
	ec2resp, err := EC2.DescribeInstances(ec2params)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	instances := []Instance{}
	for _, reservation := range ec2resp.Reservations {
		for _, instance := range reservation.Instances {
			if *instance.State.Name == "running" {

				i := Instance{
					InstanceID:     *instance.InstanceId,
					ImageID:        *instance.ImageId,
					State:          *instance.State.Name,
					InstanceType:   *instance.InstanceType,
					LaunchDatetime: instance.LaunchTime.Format(ISO_8601),
					PrivateIP:      *instance.PrivateIpAddress,
					PublicIP:       *instance.PublicIpAddress,
				}

				s := ServerVersion{}
				url := getServiceForInstance(i, "server_version")
				resp, err := apiRequest(url, "GET", nil)
				if err == nil && resp.StatusCode == 200 {
					parseBody(resp.Body, &s)
					i.GitCommitHash = s.GitCommitHash
					i.StartTime = s.StartTime
					i.Hostname = s.Hostname
				}

				instances = append(instances, i)
			}
		}
	}

	return instances, nil
}

func getAutoScaleGroup(instanceId string) (*Group, error) {

	params := &autoscaling.DescribeAutoScalingInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceId),
		},
	}

	resp, err := ASG.DescribeAutoScalingInstances(params)
	if err != nil {
		return nil, err
	}

	if len(resp.AutoScalingInstances) == 0 {
		return nil, errors.New("autoscale group for this instance not found")
	}

	name := resp.AutoScalingInstances[0].AutoScalingGroupName
	paramsAsg := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{name},
	}

	respASG, err := ASG.DescribeAutoScalingGroups(paramsAsg)
	if err != nil {
		return nil, err
	}

	if len(respASG.AutoScalingGroups) < 1 {
		return nil, errors.New("asg not found")
	}

	asg := respASG.AutoScalingGroups[0]
	launchConfigName := *asg.LaunchConfigurationName

	return &Group{
		Name:                *name,
		LaunchConfiguration: LaunchConfig{Name: launchConfigName},
		instances:           asg.Instances,
	}, nil
}

func getServiceData() (*ServiceData, error) {

	masterGitHash, err := getMasterGitHash(gConfig.RepoUrl)
	if err != nil {
		return nil, err
	}

	instanceId := gInstanceId
	autoScaleGroup, err := getAutoScaleGroup(instanceId)
	if err != nil {
		return nil, err
	}

	launchConfig, err := getLaunchConfiguration(&autoScaleGroup.LaunchConfiguration.Name)
	if err != nil {
		return nil, err
	}

	userDataDecoded, err := base64.StdEncoding.DecodeString(*launchConfig.UserData)
	if err != nil {
		printf("unable to decode user data")
		return nil, err
	}

	launchConfiguration := LaunchConfig{
		Name:     *launchConfig.LaunchConfigurationName,
		ImageID:  *launchConfig.ImageId,
		UserData: string(userDataDecoded),
	}

	group := Group{
		Name:                autoScaleGroup.Name,
		LaunchConfiguration: launchConfiguration,
	}

	instanceIds := []*string{}
	for _, instance := range autoScaleGroup.Instances() {
		instanceIds = append(instanceIds, instance.InstanceId)
	}

	instances, err := getInstances(instanceIds)
	if err != nil {
		return nil, err
	}

	return &ServiceData{
		MasterGitHash:  masterGitHash,
		InstanceID:     instanceId,
		AutoScaleGroup: group,
		InstanceList:   instances,
	}, nil

}

func getInstanceData() {

	// we can cache all this since its never going to change
	s := session.Must(session.NewSession())
	ec2Meta = ec2metadata.New(s)

	gRegion = getRegion()
	gInstanceId, _ = getInstanceId()
	gUserData = getUserData()

}

func getRegion() string {

	region, err := ec2Meta.Region()
	if err != nil {
		printf("failed to get region")
	}

	return region
}

func getUserData() string {

	userData, err := ec2Meta.GetUserData()
	if err != nil {
		printf("failed to get user data from instance")
		return ""
	}
	return userData
}

func getInstanceId() (string, error) {
	// TODO Update to use metadata
	const instanceIdUrl = "http://169.254.169.254/latest/meta-data/instance-id"
	resp, err := http.Get(instanceIdUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)

	return string(data), err
}

func getLaunchConfiguration(name *string) (*autoscaling.LaunchConfiguration, error) {

	params := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{
			name,
		},
	}
	resp, err := ASG.DescribeLaunchConfigurations(params)
	if err != nil {
		return nil, err
	}

	if len(resp.LaunchConfigurations) == 0 {
		return nil, errors.New("lc not found")
	}

	return resp.LaunchConfigurations[0], nil
}

func updateAutoscaleGroup(newHash, asgName, launchConfigName string) error {

	launchConfig, err := getLaunchConfiguration(&launchConfigName)
	if err != nil {
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(*launchConfig.UserData)
	if err != nil {
		printf("unable to decode")
		return err
	}

	newUserData := bytes.Buffer{}
	udr := bufio.NewReader(bytes.NewReader(decoded))
	for {
		line, _, err := udr.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			print("error reading userData")
			break
		}

		if strings.Contains(string(line), "GO_GIT_HASH") {
			newHashLine := fmt.Sprintf("GO_GIT_HASH=%s\n", newHash)
			newUserData.WriteString(newHashLine)
		} else {
			newUserData.Write(line)
			newUserData.Write([]byte("\n"))
		}

	}

	newUserDataEncoded := base64.StdEncoding.EncodeToString(newUserData.Bytes())
	lcNewName := ""

	groups := lcRegex.FindAllStringSubmatch(launchConfigName, -1)
	if len(groups) > 0 && len(groups[0]) > 2 {
		prefix := groups[0][1]
		iter, err := strconv.Atoi(groups[0][2])
		if err != nil {
			lcNewName = launchConfigName + "-1"
		} else {
			lcNewName = fmt.Sprintf("%s-%d", prefix, iter+1)
		}

	} else {
		lcNewName = launchConfigName + "-1"
	}

	var keyName *string = nil
	newConfig := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:      aws.String(lcNewName),
		AssociatePublicIpAddress:     aws.Bool(true),
		BlockDeviceMappings:          launchConfig.BlockDeviceMappings,
		ClassicLinkVPCId:             launchConfig.ClassicLinkVPCId,
		ClassicLinkVPCSecurityGroups: launchConfig.ClassicLinkVPCSecurityGroups,
		EbsOptimized:                 launchConfig.EbsOptimized,
		IamInstanceProfile:           launchConfig.IamInstanceProfile,
		ImageId:                      launchConfig.ImageId,
		InstanceType:                 launchConfig.InstanceType,
		InstanceMonitoring:           launchConfig.InstanceMonitoring,
		SecurityGroups:               launchConfig.SecurityGroups,
		KeyName:                      keyName,
		UserData:                     aws.String(newUserDataEncoded),
	}

	_, err = ASG.CreateLaunchConfiguration(newConfig)
	if err != nil {
		return err
	}

	asgParams := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName:    aws.String(asgName),
		LaunchConfigurationName: aws.String(lcNewName),
	}

	_, err = ASG.UpdateAutoScalingGroup(asgParams)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func ToJson(s interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(s)
	return buf.Bytes(), err
}

func ToJsonString(s interface{}) (string, error) {
	b, err := ToJson(s)
	return string(b), err
}

func printPipes(logPostFix string, stdout, stderr io.Reader) {

	stdoutpath := "/tmp/servercontrol.out.log"
	stderrpath := "/tmp/servercontrol.err.log"

	if logPostFix != "" {
		stdoutpath = stdoutpath + "." + logPostFix
		stderrpath = stderrpath + "." + logPostFix
	}

	stdoutFile, err := os.Create(stdoutpath)
	if err != nil {
		printf("unable to open tmp file for output")
		return
	}

	stderrFile, err := os.Create(stderrpath)
	if err != nil {
		printf("unable to open tmp file for output")
		return
	}

	go func() {
		defer stdoutFile.Close()
		io.Copy(stdoutFile, stdout)
	}()

	go func() {
		defer stderrFile.Close()
		io.Copy(stderrFile, stderr)
	}()

}

func getMasterGitHash(remote string) (string, error) {
	cmd := "git ls-remote " + remote + " refs/heads/master | cut -f 1 | tr -d '\n'"
	out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	return string(out), err
}

func runCommand(logPostFix string, app string, args ...string) error {

	cmd := exec.Command(app, args...)
	cmd.Dir = gConfig.RepoDir
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	printPipes(logPostFix, stdout, stderr)

	err := cmd.Start()
	if err != nil {
		fatalf(err.Error())
	}

	if err = cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			printf("exit status != 0: %s", exiterr)
			return errors.New("cmd terminated with non-zero")
		}
	}

	if !cmd.ProcessState.Success() {
		printf("exit status != 0")
		return errors.New("cmd terminated with non-zero")
	}

	return nil
}

func printf(format string, args ...interface{}) {
	if logger != nil {
		logger.Printf(format, args...)
	} else if DEBUG {
		log.Printf(format, args...)
	}
}

func fatalf(format string, args ...interface{}) {
	if logger != nil {
		logger.Fatalf(format, args...)
	} else if DEBUG {
		log.Fatalf(format, args...)
	}
}
