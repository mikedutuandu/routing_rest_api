package config

import (
	"github.com/tkanos/gonfig"
	"path"
	"path/filepath"
	"runtime"
)

type Configuration struct {
	RUN_PORT    string
	DB_USERNAME string
	DB_PASSWORD string
	DB_PORT     string
	DB_HOST     string
	DB_NAME     string

	SERVER_HOST string

	AWS_BUCKET            string
	AWS_REGION            string
	AWS_ACCESS_KEY_ID     string
	AWS_SECRET_ACCESS_KEY string
}

func GetConfig() Configuration {
	configuration := Configuration{}
	_, dirname, _, _ := runtime.Caller(0)
	filePath := path.Join(filepath.Dir(dirname), "config.json")
	gonfig.GetConf(filePath, &configuration)
	return configuration
}
