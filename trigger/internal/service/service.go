package service

import (
	"context"
	"database/sql"

	"git.yunify.com/quanxiang/trigger/apis"
	tgr "git.yunify.com/quanxiang/trigger/pkg/trigger"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"github.com/go-kit/log"
)

type service struct {
	triggerService apis.TriggerService
}

func NewServer(ctx context.Context, trigger tgr.Interface, opts ...Option) (apis.Service, error) {
	svc := &service{}

	var err error
	svc.triggerService, err = NewTrigger(ctx, trigger)
	if err != nil {
		if err != nil {
			err = errors.Wrap(err, "fail init pipeline run")
			return nil, err
		}
	}

	for _, opt := range opts {
		opt(svc.triggerService)
	}

	go trigger.Run(ctx)
	return svc, nil
}

func (p *service) GetTrigger() apis.TriggerService {
	return p.triggerService
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
