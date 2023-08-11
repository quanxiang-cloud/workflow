package main

import (
	"context"
	"flag"
	"fmt"
	triggerclient "git.yunify.com/quanxiang/trigger/pkg/client"
	"git.yunify.com/quanxiang/workflow/internal/common"
	"git.yunify.com/quanxiang/workflow/pkg/client/clientset/versioned"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"git.yunify.com/quanxiang/workflow/pkg/log"
	"git.yunify.com/quanxiang/workflow/pkg/mid/apis"
	"github.com/go-kit/log/level"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"time"
)

var configPath string

type Config struct {
	common.Config    `yaml:",inline"`
	WorkFlowInstance string `yaml:"workflow_instance"`
	TriggerInstance  string `yaml:"trigger_instance"`
	HomeHost         string `yaml:"home_host"`
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

func main() {
	flag.StringVar(&configPath, "c", "./config.yaml", "-c config path")
	flag.Parse()
	conf, err := GetConfig(configPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	logger := log.NewLogger(log.LogLevelDebug)
	ctx := context.Background()
	client := versioned.New(conf.WorkFlowInstance, logger)
	trigger := triggerclient.New(conf.TriggerInstance, logger)
	h := apis.NewHTTPHandler(logger, client, trigger, conf.Mysql, conf.WorkFlowInstance, conf.HomeHost)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.Port),
		Handler: h,
	}

	go func() {
		<-ctx.Done()
		level.Info(logger).Log("message", "Shutting down")
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Second*5,
		)
		defer cancel()
		server.Shutdown(shutdownCtx) // nolint: errcheck
	}()

	level.Info(logger).Log("message", "Starting...", "port", conf.Port)
	err = server.ListenAndServe()
	if err != http.ErrServerClosed {
		level.Error(logger).Log("message", err.Error())
	}
}
