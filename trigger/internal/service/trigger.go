package service

import (
	"context"
	"database/sql"
	"github.com/quanxiang-cloud/cabin/time"

	"git.yunify.com/quanxiang/trigger/apis"
	"git.yunify.com/quanxiang/trigger/internal/database"
	"git.yunify.com/quanxiang/trigger/internal/database/mysql"
	tgr "git.yunify.com/quanxiang/trigger/pkg/trigger"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type trigger struct {
	logger log.Logger

	trigger tgr.Interface

	triggerRepo database.TriggerRepo
}

func NewTrigger(ctx context.Context, instance tgr.Interface) (apis.TriggerService, error) {
	return &trigger{
		trigger: instance,
	}, nil
}

func (t *trigger) SetDB(db *sql.DB) {
	t.triggerRepo = mysql.NewTrigger(db)
}

func (t *trigger) SetLogger(logger log.Logger) {
	t.logger = logger
}

func (t *trigger) Create(ctx context.Context, in *apis.CreateTrigger) error {
	err := t.trigger.Add(ctx, in.Name, in.PipelineName, in.Data)
	if err != nil {
		level.Error(t.logger).Log("message", err, "name", in.Name)
		return errors.Wrap(err, "fail add trigger to instance")
	}

	err = t.triggerRepo.Create(ctx, &database.Trigger{
		Name:         in.Name,
		PipelineName: in.PipelineName,
		Type:         string(in.Type),
		Data:         in.Data,
		CreatedAt:    time.NowUnix(),
	})
	if err != nil {
		level.Error(t.logger).Log("message", err, "name", in.Name)
		return errors.Wrap(err, "fail create trigger")
	}

	return nil
}

func (t *trigger) Delete(ctx context.Context, in *apis.DeleteTrigger) error {
	err := t.trigger.Remove(ctx, in.Name)
	if err != nil {
		level.Error(t.logger).Log("message", err, "name", in.Name)
		return errors.Wrap(err, "fail remove trigger from instance")
	}

	err = t.triggerRepo.Delete(ctx, in.Name)
	if err != nil {
		level.Error(t.logger).Log("message", err, "name", in.Name)
		return errors.Wrap(err, "fail delete trigger")
	}

	return nil
}

func (t *trigger) GetDataType(ctx context.Context) tgr.Data {
	return t.trigger.GetDataType(ctx)
}
