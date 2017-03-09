// THIS FILE IS AUTOMATICALLY GENERATED. DO NOT EDIT.

// Package kinesisiface provides an interface to enable mocking the Amazon Kinesis service client
// for testing your code.
//
// It is important to note that this interface will have breaking changes
// when the service model is updated and adds new API operations, paginators,
// and waiters.
package kinesisiface

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

// KinesisAPI provides an interface to enable mocking the
// kinesis.Kinesis service client's API operation,
// paginators, and waiters. This make unit testing your code that calls out
// to the SDK's service client's calls easier.
//
// The best way to use this interface is so the SDK's service client's calls
// can be stubbed out for unit testing your code with the SDK without needing
// to inject custom request handlers into the the SDK's request pipeline.
//
//    // myFunc uses an SDK service client to make a request to
//    // Amazon Kinesis.
//    func myFunc(svc kinesisiface.KinesisAPI) bool {
//        // Make svc.AddTagsToStream request
//    }
//
//    func main() {
//        sess := session.New()
//        svc := kinesis.New(sess)
//
//        myFunc(svc)
//    }
//
// In your _test.go file:
//
//    // Define a mock struct to be used in your unit tests of myFunc.
//    type mockKinesisClient struct {
//        kinesisiface.KinesisAPI
//    }
//    func (m *mockKinesisClient) AddTagsToStream(input *kinesis.AddTagsToStreamInput) (*kinesis.AddTagsToStreamOutput, error) {
//        // mock response/functionality
//    }
//
//    func TestMyFunc(t *testing.T) {
//        // Setup Test
//        mockSvc := &mockKinesisClient{}
//
//        myfunc(mockSvc)
//
//        // Verify myFunc's functionality
//    }
//
// It is important to note that this interface will have breaking changes
// when the service model is updated and adds new API operations, paginators,
// and waiters. Its suggested to use the pattern above for testing, or using
// tooling to generate mocks to satisfy the interfaces.
type KinesisAPI interface {
	AddTagsToStreamRequest(*kinesis.AddTagsToStreamInput) (*request.Request, *kinesis.AddTagsToStreamOutput)

	AddTagsToStream(*kinesis.AddTagsToStreamInput) (*kinesis.AddTagsToStreamOutput, error)

	CreateStreamRequest(*kinesis.CreateStreamInput) (*request.Request, *kinesis.CreateStreamOutput)

	CreateStream(*kinesis.CreateStreamInput) (*kinesis.CreateStreamOutput, error)

	DecreaseStreamRetentionPeriodRequest(*kinesis.DecreaseStreamRetentionPeriodInput) (*request.Request, *kinesis.DecreaseStreamRetentionPeriodOutput)

	DecreaseStreamRetentionPeriod(*kinesis.DecreaseStreamRetentionPeriodInput) (*kinesis.DecreaseStreamRetentionPeriodOutput, error)

	DeleteStreamRequest(*kinesis.DeleteStreamInput) (*request.Request, *kinesis.DeleteStreamOutput)

	DeleteStream(*kinesis.DeleteStreamInput) (*kinesis.DeleteStreamOutput, error)

	DescribeLimitsRequest(*kinesis.DescribeLimitsInput) (*request.Request, *kinesis.DescribeLimitsOutput)

	DescribeLimits(*kinesis.DescribeLimitsInput) (*kinesis.DescribeLimitsOutput, error)

	DescribeStreamRequest(*kinesis.DescribeStreamInput) (*request.Request, *kinesis.DescribeStreamOutput)

	DescribeStream(*kinesis.DescribeStreamInput) (*kinesis.DescribeStreamOutput, error)

	DescribeStreamPages(*kinesis.DescribeStreamInput, func(*kinesis.DescribeStreamOutput, bool) bool) error

	DisableEnhancedMonitoringRequest(*kinesis.DisableEnhancedMonitoringInput) (*request.Request, *kinesis.EnhancedMonitoringOutput)

	DisableEnhancedMonitoring(*kinesis.DisableEnhancedMonitoringInput) (*kinesis.EnhancedMonitoringOutput, error)

	EnableEnhancedMonitoringRequest(*kinesis.EnableEnhancedMonitoringInput) (*request.Request, *kinesis.EnhancedMonitoringOutput)

	EnableEnhancedMonitoring(*kinesis.EnableEnhancedMonitoringInput) (*kinesis.EnhancedMonitoringOutput, error)

	GetRecordsRequest(*kinesis.GetRecordsInput) (*request.Request, *kinesis.GetRecordsOutput)

	GetRecords(*kinesis.GetRecordsInput) (*kinesis.GetRecordsOutput, error)

	GetShardIteratorRequest(*kinesis.GetShardIteratorInput) (*request.Request, *kinesis.GetShardIteratorOutput)

	GetShardIterator(*kinesis.GetShardIteratorInput) (*kinesis.GetShardIteratorOutput, error)

	IncreaseStreamRetentionPeriodRequest(*kinesis.IncreaseStreamRetentionPeriodInput) (*request.Request, *kinesis.IncreaseStreamRetentionPeriodOutput)

	IncreaseStreamRetentionPeriod(*kinesis.IncreaseStreamRetentionPeriodInput) (*kinesis.IncreaseStreamRetentionPeriodOutput, error)

	ListStreamsRequest(*kinesis.ListStreamsInput) (*request.Request, *kinesis.ListStreamsOutput)

	ListStreams(*kinesis.ListStreamsInput) (*kinesis.ListStreamsOutput, error)

	ListStreamsPages(*kinesis.ListStreamsInput, func(*kinesis.ListStreamsOutput, bool) bool) error

	ListTagsForStreamRequest(*kinesis.ListTagsForStreamInput) (*request.Request, *kinesis.ListTagsForStreamOutput)

	ListTagsForStream(*kinesis.ListTagsForStreamInput) (*kinesis.ListTagsForStreamOutput, error)

	MergeShardsRequest(*kinesis.MergeShardsInput) (*request.Request, *kinesis.MergeShardsOutput)

	MergeShards(*kinesis.MergeShardsInput) (*kinesis.MergeShardsOutput, error)

	PutRecordRequest(*kinesis.PutRecordInput) (*request.Request, *kinesis.PutRecordOutput)

	PutRecord(*kinesis.PutRecordInput) (*kinesis.PutRecordOutput, error)

	PutRecordsRequest(*kinesis.PutRecordsInput) (*request.Request, *kinesis.PutRecordsOutput)

	PutRecords(*kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error)

	RemoveTagsFromStreamRequest(*kinesis.RemoveTagsFromStreamInput) (*request.Request, *kinesis.RemoveTagsFromStreamOutput)

	RemoveTagsFromStream(*kinesis.RemoveTagsFromStreamInput) (*kinesis.RemoveTagsFromStreamOutput, error)

	SplitShardRequest(*kinesis.SplitShardInput) (*request.Request, *kinesis.SplitShardOutput)

	SplitShard(*kinesis.SplitShardInput) (*kinesis.SplitShardOutput, error)

	UpdateShardCountRequest(*kinesis.UpdateShardCountInput) (*request.Request, *kinesis.UpdateShardCountOutput)

	UpdateShardCount(*kinesis.UpdateShardCountInput) (*kinesis.UpdateShardCountOutput, error)

	WaitUntilStreamExists(*kinesis.DescribeStreamInput) error
}

var _ KinesisAPI = (*kinesis.Kinesis)(nil)
