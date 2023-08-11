package apis

import (
	"context"

	tgr "git.yunify.com/quanxiang/trigger/pkg/trigger"
)

type TriggerService interface {
	Create(ctx context.Context, in *CreateTrigger) error
	Delete(ctx context.Context, in *DeleteTrigger) error

	GetDataType(ctx context.Context) tgr.Data
}

type Type string

type CreateTrigger struct {
	Name         string   `json:"name,omitempty"`
	PipelineName string   `json:"pipeline_name,omitempty"`
	Type         Type     `json:"type,omitempty"`
	Data         tgr.Data `json:"data,omitempty"`
}

type DeleteTrigger struct {
	Name string `json:"name,omitempty"`
}

type Service interface {
	GetTrigger() TriggerService
}
