package common

import (
	"os"

	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`

	Mysql Mysql `yaml:"mysql"`

	Parallel int `yaml:"parallel"`

	Nodes []Node `yaml:"nodes"`

	Retarder struct {
		Enable     bool  `yaml:"enable"`
		BufferSize int64 `yaml:"buffer_size"`
		Delay      int64 `yaml:"delay"`
	} `yaml:"retarder"`
}

type Postgres struct {
	Host           string `yaml:"host"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	Port           int    `yaml:"port"`
	ConnectTimeout int    `yaml:"connect_timeout"`
	Database       string `yaml:"database"`
	SSLMode        string `yaml:"ssl_mode"`
}

// Mysql TODO 后期丢弃mysql改为postgres
type Mysql struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"db"`
	Log      bool   `yaml:"log"`
}
type Node struct {
	Type string   `json:"type,omitempty"`
	Host []string `json:"host,omitempty"`
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
