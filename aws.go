package aws

import (
	"log"

	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("config")
	viper.AddConfigPath("/Users/geoffpitfield/Dropbox/Development/go/src/github.com/gpitfield/aws")
	err := viper.ReadInConfig()
	if err != nil {
		log.Println(err)
	}
}
