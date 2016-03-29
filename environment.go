// Package aws provides a thin wrapper around a subset of AWS services, including instance tags
// and SQS.
package aws

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var timeout = time.Second * 10

// Set the timeout for network requests
func SetTimeout(seconds float64) {
	timeout = time.Second * time.Duration(seconds)
}

// Get the timeout for network requests
func GetTimeout() time.Duration {
	return timeout
}

// Return the instance ID for the host EC2 machine
func InstanceID() string {
	c := http.Client{Timeout: timeout}
	resp, err := c.Get("http://169.254.169.254/latest/meta-data/instance-id")
	if err != nil {
		log.Println(err)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	return string(body)
}

// Return requested tags for the given EC2 instanceID
func GetInstanceTags(instanceID string, tags []*string, region string) (results []*ec2.TagDescription, err error) {
	if region == "" {
		region = "us-east-1"
	}
	sess := session.New(&aws.Config{Region: aws.String(region)})
	svc := ec2.New(sess)
	params := &ec2.DescribeTagsInput{
		DryRun: aws.Bool(false),
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("key"),
				Values: tags,
			},
			{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(instanceID),
				},
			},
			{
				Name: aws.String("resource-type"),
				Values: []*string{
					aws.String("instance"),
				},
			},
		},
		MaxResults: aws.Int64(10), // NB is error if <5
	}
	resp, err := svc.DescribeTags(params)
	if err != nil {
		return
	}
	return resp.Tags, err
}

// Get the deployment environment for the EC2 host
func GetDeploy() (deploy string) {
	// try instanceID
	instanceID := InstanceID()
	if instanceID == "" {
		deploy = "dev"
		log.Println("environment from default", deploy)
		return
	}
	// otherwise try instance Tags
	tags, err := GetInstanceTags(instanceID, []*string{aws.String("deploy")}, "us-east-1")
	if err != nil {
		log.Println(err)
		return
	}

	if len(tags) == 1 {
		deploy = *tags[0].Value
	}
	return
}
