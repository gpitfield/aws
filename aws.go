package aws

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/spf13/viper"
)

var v *viper.Viper

func init() {
	v = viper.New()
	v.AutomaticEnv()
}

func region() string {
	return viper.GetString("region")
}

func accessKey() string {
	return viper.GetString("aws_access_key_id")
}

func secretKey() string {
	return viper.GetString("aws_secret_access_key")
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
