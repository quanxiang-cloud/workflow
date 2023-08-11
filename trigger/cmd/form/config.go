package main

import (
	"os"
	"time"

	"git.yunify.com/quanxiang/trigger/internal/database/mysql"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`

	WorkflowInstance string `yaml:"workflow_instance"`
	NVl              NVl    `yaml:"nvl"`

	MySQL mysql.MySQL `yaml:"mysql"`
}

type NVl struct {
	Enable       bool          `yaml:"enable"`
	BufferLength int           `yaml:"buffer_length"`
	TTL          time.Duration `yaml:"ttl"`
}

func GetConfig(path string) (*Config, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "fail get config file")
	}

	conf := &Config{}
	err = yaml.Unmarshal(body, conf)
	if err != nil {
		return nil, errors.Wrap(err, "fail unmarshal config file")
	}

	return conf, nil
}
