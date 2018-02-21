package servercontrol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

type ServerControlConfig struct {
	AppName      string
	RepoUrl      string
	RepoDir      string
	ServicePort  int
	Secret       string
	Prefix       string
	UpdateScript string
	Version      string
	Proto        string
	Timeout      int
}

type ServerVersion struct {
	StartTime     string `json:"start_time"`
	GitCommitHash string `json:"git_commit_hash"`
	Hostname      string `json:"hostname,omitemtpy"`
}

var (
	GitHash       = "not-set"
	startupuptime = "not-set"
	sv            ServerVersion
	gConfig       ServerControlConfig
)

func init() {
	getInstanceData()
}

func NewServerControl(config ServerControlConfig) http.Handler {

	sess = session.Must(session.NewSession(&aws.Config{
		Region: aws.String(gRegion),
	}))

	ASG = autoscaling.New(sess)
	EC2 = ec2.New(sess)

	sv.StartTime = time.Now().Format(ISO_8601)
	if h, err := os.Hostname(); err == nil {
		sv.Hostname = h
	}

	if config.RepoDir == "" {
		log.Fatal("config dir not setup")
	}

	if config.UpdateScript == "" {
		config.UpdateScript = filepath.Join(config.RepoDir, "vendor/github.com/rem7/servercontrol/git_update_to_hash.sh")
	}

	if config.Version == "" {
		config.Version = GitHash
	}

	if config.Proto == "" {
		config.Proto = "http"
	}

	if config.ServicePort == 0 {
		config.ServicePort = 80
	}

	if config.Prefix == "" {
		config.Prefix = "/server-control"
	}

	if config.Timeout == 0 {
		config.Timeout = 60
	}

	gConfig = config
	sv.GitCommitHash = config.Version

	router := mux.NewRouter().PathPrefix(config.Prefix).Subrouter().StrictSlash(true)
	router.HandleFunc("/service_data", serviceData)
	router.HandleFunc("/update_service", updateService)
	router.HandleFunc("/server_version", serverVersion)
	router.HandleFunc("/update_server", updateServer)

	router.HandleFunc("/prime_build", primeBuild)
	router.HandleFunc("/restart_server", restartServer)

	n := negroni.New()
	n.Use(negroni.HandlerFunc(auth(config)))

	n.UseHandler(router)

	return n
}

func primeBuild(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
	res.Header().Add("Content-Type", "application/json")

	props, err := parseDefaultProps(req, res)
	if err != nil {
		return
	}

	if props.Hash == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, err := os.Stat("/tmp/" + gConfig.AppName + "-" + props.Hash); err == nil {
		fmt.Fprintf(res, "binary for hash %s already exists skipping compile")
		log.Printf("build succesfull")
		return
	}

	err = internalUpdateServer(props.Hash)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(res, "pull/compiled failed")
	} else {
		fmt.Fprint(res, "build succesfull")
		log.Printf("build succesfull")
	}

}

func serverVersion(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
	res.Header().Add("Content-Type", "application/json")

	if j, err := ToJsonString(sv); err == nil {
		fmt.Fprintf(res, j)
	} else {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "%s", err.Error())
	}
}

func internalUpdateServer(hash string) error {
	return runCommand(hash, gConfig.UpdateScript, gConfig.AppName, hash)
}

func updateServer(res http.ResponseWriter, req *http.Request) {
	props, err := parseDefaultProps(req, res)
	if err != nil {
		return
	}

	if props.Hash == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	err = internalUpdateServer(props.Hash)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(res, "pull failed")
	} else {
		fmt.Fprint(res, "restarting server")
		log.Printf("pull succesfull restarting server")
		time.AfterFunc(time.Millisecond*100, func() {
			os.Exit(0)
		})
	}

}

func serviceData(res http.ResponseWriter, req *http.Request) {

	res.Header().Add("Content-Type", "application/json")

	data, err := getServiceData()
	if err != nil {
		log.Printf(err.Error())
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "%s", err.Error())
		return
	}

	json, err := ToJsonString(data)
	if err != nil {
		fmt.Fprintf(res, "%s", err.Error())
	} else {
		fmt.Fprint(res, json)
	}

}

func updateService(res http.ResponseWriter, req *http.Request) {

	props, err := parseDefaultProps(req, res)
	if err != nil {
		fmt.Fprintf(res, "%s", err.Error())
		return
	}

	if props.Hash == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err := getServiceData()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "%s", err.Error())
		return
	}

	type primeBuildJob struct {
		Err      error
		Instance Instance
	}

	// issue a build on all instances including this one
	done := make(chan primeBuildJob)
	for _, instance := range data.InstanceList {
		go func(i Instance) {
			err = primeBuildInstance(props.Hash, i)
			done <- primeBuildJob{err, i}
		}(instance)
	}

	finishedWithErrors := false
	for i := 0; i < len(data.InstanceList); i++ {
		job := <-done
		if job.Err != nil {
			finishedWithErrors = true
			log.Printf("instance %s failed to pull/compiles", job.Instance.InstanceID)
			fmt.Fprintf(res, "instance %s failed to pull/compiles", job.Instance.InstanceID)
		} else {
			log.Printf("instance %s completed build", job.Instance.InstanceID)
		}
	}

	if finishedWithErrors {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "finished with errors")
		return
	}

	// rolling restart all except this one
	for _, instance := range data.InstanceList {
		if instance.InstanceID != gInstanceId {
			err := restartServerRequest(props.Hash, instance)
			if err != nil {
				log.Printf("%v", err)
				fmt.Fprintf(res, "failed restarting server\n%s", err.Error())
				return
			}
		}
	}

	err = installVersion(props.Hash)
	if err != nil {
		msg := "unable to install version on this server"
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "%s\n%s", msg, err.Error())
		log.Printf(msg)
		return
	}

	err = updateAutoscaleGroup(props.Hash, data.AutoScaleGroup.Name, data.AutoScaleGroup.LaunchConfiguration.Name)
	if err != nil {
		log.Printf("%v", err)
		fmt.Fprintf(res, "failed updating asg/lc\n%s", err.Error())
		return
	}

	res.WriteHeader(http.StatusOK)
	fmt.Fprint(res, "Successful updating all servers, restarting this server.")
	time.AfterFunc(time.Millisecond*50, func() {
		os.Exit(0)
	})

}

func restartServer(res http.ResponseWriter, req *http.Request) {
	// install new version and restart server

	props, err := parseDefaultProps(req, res)
	if err != nil {
		fmt.Fprintf(res, "props parse failed\n%s", err.Error())
		return
	}

	if props.Hash == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	err = installVersion(props.Hash)
	if err != nil {
		log.Printf(err.Error())
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "restarted server %s with git_hash %s", gInstanceId, props.Hash)
		return
	}

	time.AfterFunc(time.Millisecond*100, func() {
		os.Exit(0)
	})

	fmt.Fprintf(res, "restarted server %s with git_hash %s", gInstanceId, props.Hash)

}

func installVersion(hash string) error {

	appname := gConfig.AppName + "-" + hash
	bin := filepath.Join("/tmp/", appname)
	dest := filepath.Join(os.Getenv("GOPATH"), "bin", gConfig.AppName)
	return runCommand("", "install", "-m", "0777", bin, dest)
}

func restartServerRequest(hash string, instance Instance) error {

	props := struct {
		Hash string `json:"hash"`
	}{
		Hash: hash,
	}

	data, _ := ToJson(props)
	url := getServiceForInstance(instance, "restart_server")

	resp, err := apiRequest(url, "GET", bytes.NewReader(data))
	if err != nil || resp.StatusCode != 200 {
		return errors.New("failed sending restart instance request")
	}

	err = waitForInstance(hash, instance)
	return err

}

func waitForInstance(hash string, instance Instance) error {

	url := getServiceForInstance(instance, "server_version")
	for i := 0; i < gConfig.Timeout; i++ {

		log.Printf("waiting for %s (%s)", instance.InstanceID, instance.PrivateIP)
		time.Sleep(1 * time.Second)

		resp, err := apiRequest(url, "GET", nil)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body := ServerVersion{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		if err != nil || resp.StatusCode != 200 {
			log.Printf("Sleeping 2 code: %v\n%v", resp.StatusCode, err)
			continue
		}

		if body.GitCommitHash == hash {
			log.Printf("updated %s (%s) with %s", instance.InstanceID, instance.PrivateIP, hash)
			return nil
		}
	}

	return errors.New("waiting on server timedout")

}

func primeBuildInstance(hash string, instance Instance) error {

	log.Printf("Updating instance: %s with %s", instance.InstanceID, hash)

	url := getServiceForInstance(instance, "prime_build")
	props := defaultProps{
		Hash:   hash,
		Secret: gConfig.Secret,
	}
	json, _ := ToJson(props)

	resp, err := apiRequest(url, "GET", bytes.NewReader(json))

	if err != nil || resp.StatusCode != 200 {
		return errors.New("failed to update instance")
	}

	return nil

}

func auth(config ServerControlConfig) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {

	return func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

		if r.Header.Get("x-sc-secret") == config.Secret {
			next(rw, r)
		} else if cookie, err := r.Cookie("secret"); err == nil && cookie.Value == config.Secret {
			next(rw, r)
		} else {
			rw.WriteHeader(http.StatusForbidden)
		}

	}

}
