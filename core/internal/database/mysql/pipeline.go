package mysql

import (
	"context"
	"database/sql"
	"encoding/json"

	"git.yunify.com/quanxiang/workflow/internal/database"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
)

type pipeline struct {
	db *sql.DB
}

func NewPipeline(db *sql.DB) database.PipelineRepo {
	return &pipeline{
		db: db,
	}
}

func (p *pipeline) Create(ctx context.Context, pl *database.Pipeline) error {
	spec, err := json.Marshal(pl.Spec)
	if err != nil {
		return errors.Wrap(err, "fail marshal pipeline spec")
	}

	row, err := p.db.Exec(
		`INSERT INTO pipeline (name, spec, created_at) VALUES (?, ?, ?) `,
		pl.Name,
		string(spec),
		pl.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "fail insert pipeline")
	}

	pl.ID, err = row.LastInsertId()
	if err != nil {
		return errors.Wrap(err, "fail get last insert id from pipeline")
	}
	return nil
}

func (p *pipeline) Update(ctx context.Context, pl *database.Pipeline) error {
	plByte, err := json.Marshal(pl.Spec)
	if err != nil {
		return errors.Wrap(err, "fail marshal pipeline spec")
	}

	_, err = p.db.ExecContext(ctx, "UPDATE pipeline SET spec = ?, updated_at =? WHERE id = ?",
		string(plByte),
		pl.UpdatedAt,
		pl.ID,
	)

	if err != nil {
		return errors.Wrap(err, "fail update pipeline")
	}
	return nil
}

func (p *pipeline) GetByName(ctx context.Context, name string) (*database.Pipeline, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT id, name, spec, created_at, updated_at FROM pipeline WHERE name = ?`,
		name,
	)

	pl := &database.Pipeline{}

	var updatedAt sql.NullInt64
	var specString sql.NullString

	err := row.Scan(&pl.ID, &pl.Name, &specString, &pl.CreatedAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "fail get pipeline")
	}

	err = json.Unmarshal([]byte(specString.String), &pl.Spec)
	if err != nil {
		return nil, errors.Wrap(err, "fail unmarhsal pipeline")
	}

	pl.UpdatedAt = updatedAt.Int64
	return pl, nil
}
