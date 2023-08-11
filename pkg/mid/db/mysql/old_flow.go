package mysql

import (
	"context"
	"database/sql"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"git.yunify.com/quanxiang/workflow/pkg/mid/db"
)

func NewOldFlow(db *sql.DB) db.OldFlowRepo {
	return &oldFlow{
		db: db,
	}
}

type oldFlow struct {
	db *sql.DB
}

func (p *oldFlow) Created(ctx context.Context, data *db.OldFlow) error {

	_, err := p.db.ExecContext(ctx,
		`INSERT INTO old_flow (id, flow_json,app_id,form_id,status,created_by) VALUES (?, ?,?,?,?,?)`,
		data.ID,
		data.FlowJson,
		data.AppID,
		data.FormID,
		data.Status,
		data.CreatedBy,
	)

	if err != nil {
		return errors.Wrap(err, "fail insert old flow")
	}

	return nil
}

func (p *oldFlow) Updated(ctx context.Context, data *db.OldFlow) error {

	_, err := p.db.ExecContext(ctx, "UPDATE old_flow SET flow_json = ?,app_id=?,form_id=? WHERE id = ?",
		data.FlowJson,
		data.AppID,
		data.FormID,
		data.ID,
	)

	if err != nil {
		return errors.Wrap(err, "fail update old flow data")
	}
	return nil
}

func (p *oldFlow) UpdatedStatus(ctx context.Context, data *db.OldFlow) error {

	_, err := p.db.ExecContext(ctx, "UPDATE old_flow SET status = ? WHERE id = ?",
		data.Status,
		data.ID,
	)

	if err != nil {
		return errors.Wrap(err, "fail update old flow status data")
	}
	return nil
}

func (p *oldFlow) Deleted(ctx context.Context, data *db.OldFlow) error {

	_, err := p.db.ExecContext(ctx, "delete from old_flow  WHERE id = ?",
		data.ID,
	)

	if err != nil {
		return errors.Wrap(err, "fail update old flow data")
	}
	return nil
}

func (p *oldFlow) Get(ctx context.Context, id string) (*db.OldFlow, error) {
	row := p.db.QueryRowContext(ctx,
		`select id,flow_json,app_id,form_id,status,created_by from old_flow  WHERE id = ?`,
		id,
	)
	flow := db.OldFlow{}
	err := row.Scan(&flow.ID, &flow.FlowJson, &flow.AppID, &flow.FormID, &flow.Status, &flow.CreatedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "fail get old flow")
	}

	return &flow, nil
}

func (p *oldFlow) Search(ctx context.Context, page, limit int, appID string) ([]db.OldFlow, int, error) {
	startIndex := (page - 1) * limit
	rows, err := p.db.QueryContext(ctx,
		`select id,flow_json,app_id,form_id,status,created_by from old_flow  WHERE app_id = ? limit ?,?`,
		appID,
		startIndex,
		limit,
	)
	if err != nil {
		return nil, 0, err
	}
	flows := make([]db.OldFlow, 0)
	for rows.Next() {
		flow := db.OldFlow{}
		err := rows.Scan(&flow.ID, &flow.FlowJson, &flow.AppID, &flow.FormID, &flow.Status, &flow.CreatedBy)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, nil
		}
		if err != nil {
			return nil, 0, errors.Wrap(err, "search get old flow")
		}
		flows = append(flows, flow)
	}
	row := p.db.QueryRowContext(ctx,
		`select count(id) as total from old_flow  WHERE app_id = ?`,
		appID,
	)
	var num int = 0
	err = row.Scan(&num)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, errors.Wrap(err, "search count old flow")
	}
	return flows, num, nil
}
