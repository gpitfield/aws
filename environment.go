package aws

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var timeout = time.Second * 10

func SetTimeout(seconds float64) {
	timeout = time.Second * time.Duration(seconds)
}

func GetTimeout() time.Duration {
	return timeout
}

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

func GetEnvironment() (environment string) {
	// try to get the variable from the environment
	environment = os.Getenv("environment")
	if environment != "" {
		log.Println("environment from envvar", environment)
		return
	}
	instanceID := InstanceID()
	if instanceID == "" {
		environment = "dev"
		log.Println("environment from default", environment)
		return
	}
	log.Println("instanceID", instanceID)
	sess := session.New(&aws.Config{Region: aws.String("us-east-1")})
	svc := ec2.New(sess)
	params := &ec2.DescribeTagsInput{
		DryRun: aws.Bool(false),
		Filters: []*ec2.Filter{
			{
				Name: aws.String("key"),
				Values: []*string{
					aws.String("environment"),
				},
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
		log.Println("describeTags", err.Error())
		return
	}

	if len(resp.Tags) == 1 {
		environment = *resp.Tags[0].Value
	}
	return
}
