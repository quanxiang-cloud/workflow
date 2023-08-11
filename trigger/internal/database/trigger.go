package database

import (
	"context"

	"git.yunify.com/quanxiang/trigger/pkg/trigger"
)

type Trigger struct {
	ID           int64
	Name         string
	PipelineName string
	Type         string
	Data         trigger.Data
	CreatedAt    int64
}

type TriggerRepo interface {
	Create(ctx context.Context, trigger *Trigger) error
	Delete(ctx context.Context, name string) error
}
