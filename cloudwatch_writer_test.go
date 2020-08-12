package zerolog2cloudwatch_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/mec07/zerolog2cloudwatch"
	"github.com/stretchr/testify/assert"
)

type mockClient struct {
	sync.Mutex
	logEvents []*cloudwatchlogs.InputLogEvent
}

func (c *mockClient) DescribeLogStreams(*cloudwatchlogs.DescribeLogStreamsInput) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	return nil, nil
}

func (c *mockClient) CreateLogGroup(*cloudwatchlogs.CreateLogGroupInput) (*cloudwatchlogs.CreateLogGroupOutput, error) {
	return nil, nil
}

func (c *mockClient) CreateLogStream(*cloudwatchlogs.CreateLogStreamInput) (*cloudwatchlogs.CreateLogStreamOutput, error) {
	return nil, nil
}

func (c *mockClient) PutLogEvents(putLogEvents *cloudwatchlogs.PutLogEventsInput) (*cloudwatchlogs.PutLogEventsOutput, error) {
	c.Lock()
	defer c.Unlock()
	if putLogEvents == nil {
		return nil, errors.New("received nil *cloudwatchlogs.PutLogEventsInput")
	}
	c.logEvents = append(c.logEvents, putLogEvents.LogEvents...)
	output := &cloudwatchlogs.PutLogEventsOutput{
		NextSequenceToken: aws.String("next sequence token"),
	}
	return output, nil
}

func (c *mockClient) getLogEvents() []*cloudwatchlogs.InputLogEvent {
	c.Lock()
	defer c.Unlock()

	fmt.Println("CLIENT LOG EVENTS: ", c.logEvents)
	logEvents := make([]*cloudwatchlogs.InputLogEvent, len(c.logEvents))
	copy(logEvents, c.logEvents)
	fmt.Println("LOG EVENTS: ", logEvents)

	return logEvents
}

type exampleLog struct {
	Time, Message, Filename string
	Port                    uint16
}

func helperWriteLogs(t *testing.T, writer io.Writer, logs ...interface{}) {
	for _, log := range logs {
		message, err := json.Marshal(log)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		_, err = writer.Write(message)
		if err != nil {
			t.Fatalf("writer.Write(%s): %v", string(message), err)
		}
	}
}

func helperMakeLogEvents(t *testing.T, logs ...interface{}) []*cloudwatchlogs.InputLogEvent {
	var logEvents []*cloudwatchlogs.InputLogEvent
	for _, log := range logs {
		message, err := json.Marshal(log)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		logEvents = append(logEvents, &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(string(message)),
			Timestamp: aws.Int64(time.Now().UTC().Unix()),
		})
	}
	return logEvents
}

// assertEqualLogMessages asserts that the log messages are all the same, ignoring the timestamps.
func assertEqualLogMessages(t *testing.T, expectedLogs []*cloudwatchlogs.InputLogEvent, logs []*cloudwatchlogs.InputLogEvent) {
	if !assert.Equal(t, len(expectedLogs), len(logs), "expected to have the same number of logs") {
		return
	}

	for index, log := range logs {
		if log == nil {
			t.Fatalf("found nil log at index: %d", index)
		}
		if expectedLogs[index] == nil {
			t.Fatalf("found nil log in expectedLogs at index: %d", index)
		}
		assert.Equal(t, *expectedLogs[index].Message, *log.Message)
	}
}

func TestCloudWatchWriter(t *testing.T) {
	client := &mockClient{}

	cloudWatchWriter, err := zerolog2cloudwatch.NewWriterWithClient(client, "logGroup", "logStream")
	if err != nil {
		t.Fatalf("NewWriterWithClient: %v", err)
	}

	aLog1 := exampleLog{
		Time:     "2009-11-10T23:00:02.043123061Z",
		Message:  "Test message 1",
		Filename: "filename",
		Port:     666,
	}
	aLog2 := exampleLog{
		Time:     "2009-11-10T23:00:02.043123061Z",
		Message:  "Test message 2",
		Filename: "filename",
		Port:     666,
	}

	helperWriteLogs(t, cloudWatchWriter, aLog1, aLog2)

	expectedLogs := helperMakeLogEvents(t, aLog1, aLog2)

	assertEqualLogMessages(t, expectedLogs, client.getLogEvents())
}
