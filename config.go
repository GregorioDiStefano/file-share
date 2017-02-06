package main

import "github.com/spf13/viper"

type config struct {
	googleCloudProjectName string
	googleBucketName       string
	googleClientEmailID    string

	mysqlUsername string
	mysqlPassword string
	mysqlHost     string

	appMaxDownloadsBeforeCaptcha int
	appMaxUploadSize             int
	appSecret                    string
}

func readConfigFile() (c *config, err error) {
	viper.SetConfigType("yaml") // or viper.SetConfigType("YAML")
	viper.SetConfigName(".config")
	viper.AddConfigPath(".")

	err = viper.ReadInConfig()

	if err != nil {
		return
	}

	c = new(config)
	c.googleCloudProjectName = viper.GetString("google.project_name")
	c.googleBucketName = viper.GetString("google.bucket_name")
	c.googleClientEmailID = viper.GetString("google.client_id")

	c.mysqlUsername = viper.GetString("mysql.user")
	c.mysqlPassword = viper.GetString("mysql.password")
	c.mysqlHost = viper.GetString("mysql.host")

	c.appMaxUploadSize = viper.GetInt("max_upload_size")
	c.appMaxDownloadsBeforeCaptcha = viper.GetInt("max_unverified_downloads")
	c.appSecret = viper.GetString("secret")

	return
}
