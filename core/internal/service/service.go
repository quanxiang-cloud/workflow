package service

import (
	"context"
	"database/sql"

	"git.yunify.com/quanxiang/workflow/apis"
	"git.yunify.com/quanxiang/workflow/internal/common"
	"git.yunify.com/quanxiang/workflow/internal/database/mysql"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type service struct {
	conf   *common.Config
	logger log.Logger

	apis.PipelineService
	apis.PipelineRunService
}

func NewServer(ctx context.Context, opts ...Option) (apis.Service, error) {
	svc := &service{}
	for _, opt := range opts {
		opt(svc)
	}

	db, err := mysql.NewDB(&svc.conf.Mysql)
	if err != nil {
		level.Error(svc.logger).Log("message", "fail connect to db", "err", err.Error())
		return nil, errors.Wrap(err, "fail connect to db")
	}
	opts = append([]Option{WithDatabase(db)}, opts...)

	svc.PipelineRunService, err = NewPipelineRunService(ctx)
	if err != nil {
		if err != nil {
			err = errors.Wrap(err, "fail init pipeline run")
			return nil, err
		}
	}

	svc.PipelineService, err = NewPipelineService(svc.PipelineRunService)
	if err != nil {
		if err != nil {
			err = errors.Wrap(err, "fail init pipeline")
			return nil, err
		}
	}

	for _, opt := range opts {
		opt(svc.PipelineService)
		opt(svc.PipelineRunService)
	}
	return svc, nil
}

func (s *service) SetConfig(conf *common.Config) {
	s.conf = conf
}

func (s *service) SetLogger(logger log.Logger) {
	s.logger = log.With(logger, "module", "service")
}

func (s *service) GetPipeline() apis.PipelineService {
	return s.PipelineService
}
func (s *service) GetPipelineRun() apis.PipelineRunService {
	return s.PipelineRunService
}

type Option func(s interface{})

type Logger interface {
	SetLogger(log.Logger)
}

func WithLogger(logger log.Logger) Option {
	return func(s interface{}) {
		if s, ok := s.(Logger); ok {
			s.SetLogger(logger)
		}
	}
}

type Config interface {
	SetConfig(*common.Config)
}

func WithConfig(conf *common.Config) Option {
	return func(s interface{}) {
		if s, ok := s.(Config); ok {
			s.SetConfig(conf)
		}
	}
}

type Database interface {
	SetDB(*sql.DB)
}

func WithDatabase(db *sql.DB) Option {
	return func(s interface{}) {
		if s, ok := s.(Database); ok {
			s.SetDB(db)
		}
	}
}
