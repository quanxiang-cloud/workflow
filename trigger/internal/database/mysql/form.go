package mysql

import (
	"context"
	"database/sql"
	"encoding/json"

	"git.yunify.com/quanxiang/trigger/internal/database"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
)

type form struct {
	db *sql.DB
}

func NewForm(db *sql.DB) database.FormRepo {
	return &form{
		db: db,
	}
}

func (f *form) Create(ctx context.Context, fl *database.FormTrigger) error {
	filters, err := json.Marshal(fl.Filters)
	if err != nil {
		return errors.Wrap(err, "fail marshal filters")
	}
	row, err := f.db.ExecContext(ctx,
		`INSERT INTO form_trigger (name,pipeline_name, app_id, table_id, type,filters, created_at)
	VALUES (?, ?, ?, ?, ?,?,?)`,
		fl.Name,
		fl.PipelineName,
		fl.AppID,
		fl.TableID,
		fl.Type,
		string(filters),
		fl.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "fail insert form trigger")
	}

	fl.ID, err = row.LastInsertId()
	if err != nil {
		return errors.Wrap(err, "fail get last insert id from form trigger")
	}
	return nil
}

func (f *form) Get(ctx context.Context, name string) (*database.FormTrigger, error) {
	row := f.db.QueryRowContext(ctx, `SELECT id, name,pipeline_name, app_id, table_id, type, filters, created_at FROM form_trigger WHERE name = ?`, name)
	if err := row.Err(); err != nil {
		return nil, errors.Wrap(err, "fail get form trigger")
	}

	var filters sql.NullString

	fl := &database.FormTrigger{}

	err := row.Scan(&fl.ID, &fl.Name, &fl.PipelineName, &fl.AppID, &fl.TableID, &fl.Type, &filters, &fl.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "fail scan form trigger")
	}

	if filters.String != "" {
		err := json.Unmarshal([]byte(filters.String), &fl.Filters)
		if err != nil {
			return nil, errors.Wrap(err, "fail unmarshal filters")
		}
	}

	return fl, nil
}

func (f *form) List(ctx context.Context) (result []*database.FormTrigger, err error) {
	rows, err := f.db.QueryContext(ctx, `SELECT id, name,pipeline_name, app_id, table_id, type, filters, created_at FROM form_trigger`)
	if err != nil {
		return nil, errors.Wrap(err, "fail get form trigger")
	}

	for rows.Next() {
		var filters sql.NullString

		fl := &database.FormTrigger{}
		err := rows.Scan(&fl.ID, &fl.Name, &fl.PipelineName, &fl.AppID, &fl.TableID, &fl.Type, &filters, &fl.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "fail scan form trigger")
		}

		if filters.String != "" {
			err := json.Unmarshal([]byte(filters.String), &fl.Filters)
			if err != nil {
				return nil, errors.Wrap(err, "fail unmarshal filters")
			}
		}

		result = append(result, fl)
	}

	return
}

func (f *form) GetByTableID(ctx context.Context, tableID, _t string) (result []*database.FormTrigger, err error) {
	rows, err := f.db.QueryContext(ctx, `SELECT id, name,pipeline_name, app_id, table_id, type, filters, created_at FROM form_trigger
	WHERE table_id = ? and type = ?`,
		tableID, _t)
	if err != nil {
		return nil, errors.Wrap(err, "fail get form trigger")
	}

	for rows.Next() {
		var filters sql.NullString
		fl := &database.FormTrigger{}
		err := rows.Scan(&fl.ID, &fl.Name, &fl.PipelineName, &fl.AppID, &fl.TableID, &fl.Type, &filters, &fl.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "fail scan form trigger")
		}

		if filters.String != "" {
			err := json.Unmarshal([]byte(filters.String), &fl.Filters)
			if err != nil {
				return nil, errors.Wrap(err, "fail unmarshal filters")
			}
		}

		result = append(result, fl)
	}

	return
}

func (f *form) Delete(ctx context.Context, id int64) error {
	_, err := f.db.ExecContext(ctx, "DELETE FROM form_trigger WHERE id = ?", id)
	if err != nil {
		return errors.Wrap(err, "fail delete from form trigger")
	}
	return nil
}
