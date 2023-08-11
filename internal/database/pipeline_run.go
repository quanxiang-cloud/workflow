package database

import (
	"context"

	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
)

type PipelineRun struct {
	ID        int64
	Pipeline  v1alpha1.Pipeline
	Spec      v1alpha1.PipeplineRunSpec
	Status    v1alpha1.PipeplineRunStatus
	State     v1alpha1.PipelineSatus
	CreatedAt int64
	UpdatedAt int64
}

type PipelineRunRepo interface {
	Create(ctx context.Context, plr *PipelineRun) error
	Update(ctx context.Context, plr *PipelineRun) error
	Get(ctx context.Context, id int64) (*PipelineRun, error)
	ListRunning(ctx context.Context) ([]int64, error)
}
