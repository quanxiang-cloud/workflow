package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"git.yunify.com/quanxiang/workflow/pkg/log"
	"git.yunify.com/quanxiang/workflow/pkg/node"
	"git.yunify.com/quanxiang/workflow/pkg/node/nodes/examine/apis"
	"git.yunify.com/quanxiang/workflow/pkg/node/nodes/examine/db/mysql"
	"git.yunify.com/quanxiang/workflow/pkg/node/nodes/examine/service"
	"gopkg.in/yaml.v3"
)

type Examine struct {
	task service.Task
}
type Config struct {
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`

	Mysql            mysql.Mysql `yaml:"mysql"`
	QxInstance       []string    `yaml:"qx_instance"`
	WorkFlowInstance string      `yaml:"work_flow_instance"`
	HomeHost         string      `yaml:"home_host"`
}

var configPath string

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
func main() {
	flag.StringVar(&configPath, "c", "./config.yaml", "-c config path")
	flag.Parse()
	conf, err := GetConfig(configPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	logger := log.NewLogger(log.LogLevelDebug)
	s := &Examine{}

	if err != nil {
		panic(err)
	}
	db, err := mysql.NewDB(&conf.Mysql)
	if err != nil {
		panic(err)
	}
	s.task = service.NewTask(db, conf.QxInstance, logger, conf.WorkFlowInstance, conf.HomeHost)
	ctx := context.Background()
	endPoints := apis.NewEndPoints(s.task)
	node.Main(logger, fmt.Sprintf(":%d", conf.Port))(ctx, endPoints, apis.Router(endPoints)...)
}
