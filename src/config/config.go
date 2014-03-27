package config

import (
	"encoding/json"
	"io/ioutil"
)

type ConfigInfo struct {
	BasePath                string
	TargetReposName         []string
	AnalysisResultReposName string
	AnalysisResultReposUrl  string
	MailSender              string
	MailScript              string
	DBDriverName            string
	DBDataSourceName        string
}

func ParseConfig(configPath string) (conf ConfigInfo, err error) {
	bytes, err := ioutil.ReadFile(configPath)
	err = json.Unmarshal(bytes, &conf)
	return
}
