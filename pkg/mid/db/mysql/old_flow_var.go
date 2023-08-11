package mysql

import (
	"context"
	"database/sql"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"git.yunify.com/quanxiang/workflow/pkg/mid/db"
)

func NewOldFlowVar(db *sql.DB) db.OldFlowVarRepo {
	return &oldFlowVar{
		db: db,
	}
}

type oldFlowVar struct {
	db *sql.DB
}

func (p *oldFlowVar) Created(ctx context.Context, data *db.OldFlowVar) error {

	_, err := p.db.ExecContext(ctx,
		"INSERT INTO old_flow_var (id, flow_id,code,default_value,`desc`,field_type,`name`,`type`,format,created_at,creator_avatar,creator_id,updated_at,updator_name,updated_id) VALUES (?, ?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		data.ID,
		data.FlowID,
		data.Code,
		data.DefaultValue,
		data.Desc,
		data.FieldType,
		data.Name,
		data.Type,
		data.Format,
		data.CreatedAt,
		data.CreatorAvatar,
		data.CreatorID,
		data.UpdatedAt,
		data.UpdatedID,
		data.UpdatorName,
	)

	if err != nil {
		return errors.Wrap(err, "fail insert old flow var")
	}

	return nil
}

func (p *oldFlowVar) Updated(ctx context.Context, data *db.OldFlowVar) error {

	_, err := p.db.ExecContext(ctx, "UPDATE old_flow_var SET  default_value=?,`desc`=?,field_type=?,`name`=?,`type`=?,format=?,updated_at=?,updator_name=?,updated_id=? WHERE id = ?",
		data.DefaultValue,
		data.Desc,
		data.FieldType,
		data.Name,
		data.Type,
		data.Format,
		data.UpdatedAt,
		data.UpdatorName,
		data.UpdatedID,
		data.ID,
	)

	if err != nil {
		return errors.Wrap(err, "fail update old flow var data")
	}
	return nil
}

func (p *oldFlowVar) Deleted(ctx context.Context, id string) error {

	_, err := p.db.ExecContext(ctx, "delete from old_flow_var  WHERE id = ?",
		id,
	)

	if err != nil {
		return errors.Wrap(err, "fail delete old flow var data")
	}
	return nil
}

func (p *oldFlowVar) Get(ctx context.Context, id string) (*db.OldFlowVar, error) {
	row := p.db.QueryRowContext(ctx,
		"select id, flow_id,code,default_value,`desc`,field_type,`name`,`type`,format,created_at,creator_avatar,creator_id,updated_at,updator_name,updated_id from old_flow_var  WHERE id = ?",
		id,
	)
	flowVar := db.OldFlowVar{}
	err := row.Scan(
		&flowVar.ID,
		&flowVar.FlowID,
		&flowVar.Code,
		&flowVar.DefaultValue,
		&flowVar.Desc,
		&flowVar.FieldType,
		&flowVar.Name,
		&flowVar.Type,
		&flowVar.Format,
		&flowVar.CreatedAt,
		&flowVar.CreatorAvatar,
		&flowVar.CreatorID,
		&flowVar.UpdatedAt,
		&flowVar.UpdatedID,
		&flowVar.UpdatorName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "fail get old flow var")
	}

	return &flowVar, nil
}

func (p *oldFlowVar) GetByFlowID(ctx context.Context, flowID string) ([]db.OldFlowVar, error) {
	rows, err := p.db.QueryContext(ctx,
		"select id, flow_id,code,default_value,`desc`,field_type,`name`,`type`,format,created_at,creator_avatar,creator_id,updated_at,updator_name,updated_id from old_flow_var  WHERE flow_id = ?",
		flowID,
	)
	if err != nil {
		return nil, err
	}
	vars := make([]db.OldFlowVar, 0)
	for rows.Next() {
		flowVar := db.OldFlowVar{}
		err := rows.Scan(
			&flowVar.ID,
			&flowVar.FlowID,
			&flowVar.Code,
			&flowVar.DefaultValue,
			&flowVar.Desc,
			&flowVar.FieldType,
			&flowVar.Name,
			&flowVar.Type,
			&flowVar.Format,
			&flowVar.CreatedAt,
			&flowVar.CreatorAvatar,
			&flowVar.CreatorID,
			&flowVar.UpdatedAt,
			&flowVar.UpdatedID,
			&flowVar.UpdatorName,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, errors.Wrap(err, "fail get old flow var by flow id")
		}
		vars = append(vars, flowVar)
	}

	return vars, nil
}

//func (p *oldFlowVar) Search(ctx context.Context, id string) (*db.OldFlowVar, error) {
//	rows,err := p.db.QueryContext(ctx,
//		`select id, flow_id,code,default_value,desc,field_type,name,type,created_at,creator_avatar,creator_id,updated_at,updator_name,updated_id from old_flow_var offset ? limit ?`,
//		id,
//	)
//	flowVar := db.OldFlowVar{}
//	err := row.Scan(
//		&flowVar.ID,
//		&flowVar.FlowID,
//		&flowVar.Code,
//		&flowVar.DefaultValue,
//		&flowVar.Desc,
//		&flowVar.FieldType,
//		&flowVar.Name,
//		&flowVar.Type,
//		&flowVar.CreatedAt,
//		&flowVar.CreatorAvatar,
//		&flowVar.CreatorID,
//		&flowVar.UpdatedAt,
//		&flowVar.UpdatedID,
//		&flowVar.UpdatorName,
//	)
//	if errors.Is(err, sql.ErrNoRows) {
//		return nil, nil
//	}
//	if err != nil {
//		return nil, errors.Wrap(err, "fail get old flow var")
//	}
//
//	return &flowVar, nil
//}
