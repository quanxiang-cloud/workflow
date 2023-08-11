package mysql

import (
	"context"
	"database/sql"

	"git.yunify.com/quanxiang/trigger/internal/database"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"github.com/goccy/go-json"
)

type trigger struct {
	db *sql.DB
}

func NewTrigger(db *sql.DB) database.TriggerRepo {
	return &trigger{
		db: db,
	}
}

func (t *trigger) Create(ctx context.Context, trigger *database.Trigger) error {
	data, err := json.Marshal(trigger.Data)
	if err != nil {
		return errors.Wrap(err, "fail marshal trigger data")
	}

	row, err := t.db.ExecContext(ctx,
		"INSERT INTO `trigger` (name, pipeline_name, type, data, created_at) VALUES (?, ?, ?, ?, ?) ",
		trigger.Name,
		trigger.PipelineName,
		trigger.Type,
		string(data),
		trigger.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "fail insert trigger")
	}

	trigger.ID, err = row.LastInsertId()
	if err != nil {
		return errors.Wrap(err, "fail get last insert id from trigger")
	}
	return nil
}
func (t *trigger) Delete(ctx context.Context, name string) error {
	_, err := t.db.ExecContext(ctx, "DELETE FROM `trigger` WHERE name = ?", name)
	if err != nil {
		return errors.Wrap(err, "fail delete a trigger")
	}
	return nil
}
