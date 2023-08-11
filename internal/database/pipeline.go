package database

import (
	"context"

	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
)

type Pipeline struct {
	ID        int64
	Name      string
	Spec      v1alpha1.PipelineSpec
	CreatedAt int64
	UpdatedAt int64
}

type PipelineRepo interface {
	Create(ctx context.Context, pl *Pipeline) error

	Update(ctx context.Context, pl *Pipeline) error

	GetByName(ctx context.Context, name string) (*Pipeline, error)
}
