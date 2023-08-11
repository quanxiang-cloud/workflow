package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"git.yunify.com/quanxiang/workflow/apis"
	"git.yunify.com/quanxiang/workflow/internal/common"
	"git.yunify.com/quanxiang/workflow/internal/service"
	"git.yunify.com/quanxiang/workflow/pkg/helper/logger"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

var configPath string

func main() {
	flag.StringVar(&configPath, "c", "./config.yaml", "-c config path")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conf, err := common.GetConfig(configPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	logger := logger.NewLogger(conf.LogLevel)

	svc, err := service.NewServer(ctx,
		service.WithLogger(logger),
		service.WithConfig(conf),
	)
	if err != nil {
		level.Error(logger).Log("message", err.Error())
		os.Exit(1)
	}

	h := apis.NewHTTPHandler(conf, svc, log.With(logger, "component", "HTTP"))
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
