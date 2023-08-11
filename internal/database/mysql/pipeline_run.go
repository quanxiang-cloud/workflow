package mysql

import (
	"context"
	"database/sql"
	"encoding/json"

	"git.yunify.com/quanxiang/workflow/internal/database"
	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
)

type pipelineRun struct {
	db *sql.DB
}

func NewPipelineRun(db *sql.DB) database.PipelineRunRepo {
	return &pipelineRun{
		db: db,
	}
}

func (p *pipelineRun) Create(ctx context.Context, plr *database.PipelineRun) error {
	plByte, err := json.Marshal(plr.Pipeline)
	if err != nil {
		return errors.Wrap(err, "fail marshal pipeline")
	}
	specByte, err := json.Marshal(plr.Spec)
	if err != nil {
		return errors.Wrap(err, "fail marshal pipeline run params")
	}
	statusByte, err := json.Marshal(plr.Status)
	if err != nil {
		return errors.Wrap(err, "fail marshal pipeline run status")
	}

	row, err := p.db.ExecContext(ctx,
		`INSERT INTO pipeline_run (pipeline, spec, status, created_at)
		VALUES (?, ?, ?, ?)`,
		string(plByte),
		string(specByte),
		string(statusByte),
		plr.CreatedAt,
	)
	if err != nil {
		return errors.Wrap(err, "fail insert pipeline run")
	}

	plr.ID, err = row.LastInsertId()
	if err != nil {
		return errors.Wrap(err, "fail get last insert id from pipeline run")
	}
	return nil
}
func (p *pipelineRun) Update(ctx context.Context, plr *database.PipelineRun) error {
	plByte, err := json.Marshal(plr.Pipeline)
	if err != nil {
		return errors.Wrap(err, "fail marshal pipeline")
	}
	specByte, err := json.Marshal(plr.Spec)
	if err != nil {
		return errors.Wrap(err, "fail marshal pipeline run params")
	}
	statusByte, err := json.Marshal(plr.Status)
	if err != nil {
		return errors.Wrap(err, "fail marshal pipeline run status")
	}

	_, err = p.db.ExecContext(ctx,
		`UPDATE pipeline_run set pipeline = ?, spec = ?, status = ?, state = ?, updated_at = ?`,
		string(plByte),
		string(specByte),
		string(statusByte),
		plr.State,
		plr.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "fail update pipeline run")
	}
	return nil
}
func (p *pipelineRun) Get(ctx context.Context, id int64) (*database.PipelineRun, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT id, pipeline, spec, status, state, created_at, updated_at FROM pipeline_run WHERE id = ?`,
		id)

	plr := &database.PipelineRun{}
	var updatedAt sql.NullInt64
	var pipeline string
	var spec string
	var status string
	var state sql.NullString

	err := row.Scan(&plr.ID, &pipeline, &spec, &status, &state, &plr.CreatedAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "fail get pipeline run")
	}

	err = json.Unmarshal([]byte(pipeline), &plr.Pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "fail unmarhsal pipeline")
	}
	err = json.Unmarshal([]byte(spec), &plr.Spec)
	if err != nil {
		return nil, errors.Wrap(err, "fail unmarhsal spec")
	}
	err = json.Unmarshal([]byte(status), &plr.Status)
	if err != nil {
		return nil, errors.Wrap(err, "fail unmarhsal status")
	}
	plr.UpdatedAt = updatedAt.Int64
	plr.State = v1alpha1.PipelineSatus(state.String)

	return plr, nil
}

func (p *pipelineRun) ListRunning(ctx context.Context) ([]int64, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id FROM pipeline_run  WHERE state != ? and state != ? `, v1alpha1.PipelineRunFinish, v1alpha1.PipelineRunKill,
	)
	if err != nil {
		return nil, errors.Wrap(err, "fail list running pipeline run")
	}

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			return nil, errors.Wrap(err, "fail scan pipeline run id")
		}
		ids = append(ids, id)
	}
	return ids, nil
}
