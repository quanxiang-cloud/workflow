package service

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"git.yunify.com/quanxiang/workflow/apis"
	"git.yunify.com/quanxiang/workflow/internal/database"
	"git.yunify.com/quanxiang/workflow/internal/database/mysql"
	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func NewPipelineService(plrs apis.PipelineRunService) (apis.PipelineService, error) {
	return &pipelineService{
		pipelineRun: plrs,
	}, nil
}

type pipelineService struct {
	logger log.Logger

	pipelineRepo database.PipelineRepo
	pipelineRun  apis.PipelineRunService
}

func (p *pipelineService) SetLogger(logger log.Logger) {
	p.logger = log.With(logger, "module", "service")
}

func (p *pipelineService) SetDB(db *sql.DB) {
	p.pipelineRepo = mysql.NewPipeline(db)
}

func (p *pipelineService) Save(ctx context.Context, in *apis.SavePipeline) error {
	pl, err := p.pipelineRepo.GetByName(ctx, in.Pipeline.Name)
	if err != nil {
		level.Error(p.logger).Log("message", err.Error())
		return errors.Wrap(err, "fail get pipeline from database")
	}
	if pl != nil {
		pl.Spec = in.Pipeline.Spec
		pl.UpdatedAt = time.Now().Unix()
		err = p.pipelineRepo.Update(ctx, pl)
		if err != nil {
			level.Error(p.logger).Log("message", err.Error())
			return errors.Wrap(err, "fail uodate pipeline to database")
		}
		return nil
	}

	//  save pipeline to DB
	err = p.pipelineRepo.Create(ctx, &database.Pipeline{
		Name:      in.Pipeline.Name,
		Spec:      in.Pipeline.Spec,
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		level.Error(p.logger).Log("message", err.Error())
		return errors.Wrap(err, "fail save pipeline to database")
	}
	return nil
}

func (p *pipelineService) Exec(ctx context.Context, in *apis.ExecPipeline) error {
	// get pipeline by name
	pipeline, err := p.pipelineRepo.GetByName(ctx, in.Name)
	if err != nil {
		level.Error(p.logger).Log("message", err.Error())
		return errors.Wrap(err, "fail get pipeline, while exec pipeline run")
	}
	if pipeline == nil {
		return errors.NewErr(http.StatusBadRequest, &errors.CodeError{
			Code:    http.StatusNotFound,
			Message: "pipeline not exists",
		})
	}

	// create pipeline run
	cplr := &apis.CreatePipelineRun{}
	cplr.Params = in.Params
	cplr.Pipeline = &v1alpha1.Pipeline{
		Name: pipeline.Name,
		Spec: pipeline.Spec,
	}
	err = p.pipelineRun.Create(ctx, cplr)
	if err != nil {
		level.Error(p.logger).Log("message", err.Error())
		return errors.Wrap(err, "fail create pipeline")
	}
	return nil
}
