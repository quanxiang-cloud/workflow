package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"git.yunify.com/quanxiang/trigger/apis"
	"git.yunify.com/quanxiang/trigger/internal/database/mysql"
	"git.yunify.com/quanxiang/trigger/internal/service"
	"git.yunify.com/quanxiang/workflow/pkg/helper/logger"
	"github.com/gin-gonic/gin"
	ll "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

var configPath string

func main() {
	flag.StringVar(&configPath, "c", "./config.yaml", "-c config path")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conf, err := GetConfig(configPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	logger := logger.NewLogger(conf.LogLevel)

	db, err := mysql.NewDB(&conf.MySQL)
	if err != nil {
		level.Error(logger).Log("message", "fail connect to db", "err", err.Error())
		os.Exit(1)
	}
	opts := []service.Option{
		service.WithLogger(logger),
		service.WithDatabase(db),
	}

	ft, err := New(ctx, conf, opts...)
	if err != nil {
		level.Error(logger).Log("message", err.Error())
		os.Exit(1)
	}

	svc, err := service.NewServer(ctx,
		ft,
		opts...,
	)
	if err != nil {
		level.Error(logger).Log("message", err.Error())
		os.Exit(1)
	}

	h := apis.NewHTTPHandler(svc, logger)
	NewServer(h, ft, logger)

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

func NewServer(defaulHandler http.Handler, svc FromService, logger ll.Logger) http.Handler {
	r := defaulHandler.(*gin.Engine)

	r.POST("/api/v1/trigger/exec", func(c *gin.Context) {
		body := &struct {
			Data Data `json:"data,omitempty"`
		}{}
		err := c.ShouldBindJSON(body)
		if err != nil {
			level.Info(logger).Log("message", err)
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		err = svc.Exec(c.Request.Context(), &body.Data)
		if err != nil {
			level.Error(logger).Log("message", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	})

	return r
}
