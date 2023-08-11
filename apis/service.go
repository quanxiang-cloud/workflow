package apis

import (
	"context"

	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
)

type SavePipeline struct {
	v1alpha1.Pipeline `json:",inline"`
}

type ExecPipeline struct {
	Name   string
	Params []*v1alpha1.KeyAndValue `json:"params,omitempty"`
}

type PipelineService interface {
	Save(ctx context.Context, in *SavePipeline) error
	Exec(ctx context.Context, in *ExecPipeline) error
}

type plr struct {
	Pipeline *v1alpha1.Pipeline      `json:"pipeline,omitempty"`
	Params   []*v1alpha1.KeyAndValue `json:"params,omitempty"`
}

type CreatePipelineRun struct {
	plr
}

type ExecPipelineRun struct {
	ID int64 `json:"id,omitempty"`
}

type PipelineRunService interface {
	Create(ctx context.Context, in *CreatePipelineRun) error
	Exec(ctx context.Context, in *ExecPipelineRun) error
}

type Service interface {
	GetPipeline() PipelineService
	GetPipelineRun() PipelineRunService
}
