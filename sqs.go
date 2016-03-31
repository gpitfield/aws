// Package aws provides a thin wrapper around a subset of AWS services, including instance tags
// and SQS. Note that this makes use of the default AWS environment variables AWS_ACCESS_KEY_ID,
// AWS_SECRET_ACCESS_KEY, and REGION
package aws

import (
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	channels = struct {
		sync.RWMutex
		m map[string]chan []byte
	}{m: make(map[string]chan []byte)}
	queueURLs = struct {
		sync.RWMutex
		m map[string]string
	}{m: make(map[string]string)}
	sqsService *sqs.SQS
	pollWait   int64 = 20
)

// Return a singleton SQS service instance
func SQS() *sqs.SQS {
	if sqsService == nil {
		if accessKey() != "" && secretKey() != "" && region() != "" {
			sqsService = sqs.New(session.New(&aws.Config{
				Region:      aws.String(region()),
				Credentials: credentials.NewStaticCredentials(accessKey(), secretKey(), ""),
			}))
		} else {
			log.Println("no AWS environment variables found; defaulting to EC2 instance profile.")
			sqsService = sqs.New(session.New())
		}
	}
	return sqsService
}

func queueURL(queue string) (url string) {
	queueURLs.RLock()
	if url, exists := queueURLs.m[queue]; !exists {
		queueURLs.RUnlock()
		params := sqs.CreateQueueInput{
			QueueName: aws.String(queue),
		}
		resp, err := SQS().CreateQueue(&params)
		if err != nil {
			fmt.Println(err)
		} else {
			queueURLs.Lock()
			queueURLs.m[queue] = *resp.QueueUrl
			queueURLs.Unlock()
		}
		return *resp.QueueUrl
	} else {
		queueURLs.RUnlock()
		return url
	}
}

// SQS channel returns a blocking go chan that masks the underlying SQS transport
func SQSChannel(queue string) (c chan []byte) {
	channels.RLock()
	if c, exists := channels.m[queue]; !exists {
		channels.RUnlock()
		channels.Lock()
		c = make(chan []byte) // blocking channel
		channels.m[queue] = c
		channels.Unlock()
		go receiveQueue(queue)
	} else {
		channels.RUnlock()
	}
	channels.RLock()
	defer channels.RUnlock()
	return channels.m[queue]
}

// Put a message on the named queue
func QueueMessage(queue string, message []byte) (err error) {
	svc := SQS()
	params := &sqs.SendMessageInput{
		MessageBody: aws.String(string(message)),
		QueueUrl:    aws.String(queueURL(queue)),
	}
	_, err = svc.SendMessage(params)
	return
}

// Receive from the named SQS queue, transferring its contents to the corresponding go chan
func receiveQueue(queue string) {
	svc := SQS()
	params := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL(queue)),
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(1),
		WaitTimeSeconds:     aws.Int64(pollWait), // long polling
	}
	for {
		resp, err := svc.ReceiveMessage(params)
		if err != nil {
			log.Println(err)
		} else if len(resp.Messages) > 0 {
			msg := resp.Messages[0]
			SQSChannel(queue) <- []byte(*msg.Body)

			// delete message
			deleteParams := &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL(queue)),
				ReceiptHandle: aws.String(*msg.ReceiptHandle),
			}
			svc.DeleteMessage(deleteParams)
		}
	}
}
