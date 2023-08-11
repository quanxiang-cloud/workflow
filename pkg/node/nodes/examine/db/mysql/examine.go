package mysql

import (
	"context"
	"database/sql"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	model "git.yunify.com/quanxiang/workflow/pkg/node/nodes/examine/db"
)

type examineNode struct {
	db *sql.DB
}

func NewExamineNode(db *sql.DB) model.TaskRepo {
	return &examineNode{
		db: db,
	}
}

func (t *examineNode) InsertBranch(ctx context.Context, datas ...*model.Task) (err error) {
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	for k := range datas {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO `examine_node` (id, task_id,flow_id,user_id,substitute,created_by,examine_type,`result`,node_result,created_at,updated_at,app_id,form_table_id,form_data_id,form_ref,urge_times,remark,node_def_key) VALUES (?, ?,?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?,?)",
			datas[k].ID,
			datas[k].TaskID,
			datas[k].FlowID,
			datas[k].UserID,
			datas[k].Substitute,
			datas[k].CreatedBy,
			datas[k].ExamineType,
			datas[k].Result,
			datas[k].NodeResult,
			datas[k].CreatedAt,
			datas[k].UpdatedAt,
			datas[k].AppID,
			datas[k].FormTableID,
			datas[k].FormDataID,
			datas[k].FormRef,
			datas[k].UrgeTimes,
			datas[k].Remark,
			datas[k].NodeDefKey,
		)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "fail insert old flow")
		}
	}

	tx.Commit()
	return nil
}

func (t *examineNode) UpdateSubstitute(ctx context.Context, data *model.Task) error {
	_, err := t.db.ExecContext(ctx, "UPDATE `examine_node` SET substitute = ?,updated_at=? WHERE id = ?",
		data.Substitute,
		data.UpdatedAt,
		data.ID,
	)
	if err != nil {
		return err
	}
	return nil
}
func (t *examineNode) UpdateResult(ctx context.Context, data *model.Task) error {
	_, err := t.db.ExecContext(ctx, "UPDATE `examine_node` SET `result` = ?,node_result=?,updated_at=?,remark=? WHERE id = ?",
		data.Result,
		data.NodeResult,
		data.UpdatedAt,
		data.Remark,
		data.ID,
	)
	if err != nil {
		return err
	}
	return nil
}
func (t *examineNode) UpdateByTaskID(ctx context.Context, data *model.Task) error {
	_, err := t.db.ExecContext(ctx, "UPDATE `examine_node` SET `node_result` = ?,updated_at=? WHERE task_id = ?",
		data.NodeResult,
		data.UpdatedAt,
		data.TaskID,
	)
	if err != nil {
		return err
	}
	return nil
}

func (t *examineNode) ListByTaskID(ctx context.Context, taskID string) ([]model.Task, error) {
	rows, err := t.db.QueryContext(ctx,
		"select id, task_id,flow_id,user_id,substitute,created_by,examine_type,`result`,node_result,created_at,updated_at,app_id,form_table_id,form_data_id,form_ref,urge_times,remark,node_def_key from examine_node  WHERE task_id = ?",
		taskID,
	)
	if err != nil {
		return nil, err
	}
	tasks := make([]model.Task, 0)
	for rows.Next() {
		task := model.Task{}
		err := rows.Scan(
			&task.ID,
			&task.TaskID,
			&task.FlowID,
			&task.UserID,
			&task.Substitute,
			&task.CreatedBy,
			&task.ExamineType,
			&task.Result,
			&task.NodeResult,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.AppID,
			&task.FormTableID,
			&task.FormDataID,
			&task.FormRef,
			&task.UrgeTimes,
			&task.Remark,
			&task.NodeDefKey,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (t *examineNode) ListByTaskIDAndNodeDefKey(ctx context.Context, taskID, nodeDefKey string) ([]model.Task, error) {
	rows, err := t.db.QueryContext(ctx,
		"select id, task_id,flow_id,user_id,substitute,created_by,examine_type,`result`,node_result,created_at,updated_at,app_id,form_table_id,form_data_id,form_ref,urge_times,remark,node_def_key from examine_node  WHERE task_id = ? and node_def_key=?",
		taskID,
		nodeDefKey,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	tasks := make([]model.Task, 0)
	for rows.Next() {
		task := model.Task{}
		err = rows.Scan(
			&task.ID,
			&task.TaskID,
			&task.FlowID,
			&task.UserID,
			&task.Substitute,
			&task.CreatedBy,
			&task.ExamineType,
			&task.Result,
			&task.NodeResult,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.AppID,
			&task.FormTableID,
			&task.FormDataID,
			&task.FormRef,
			&task.UrgeTimes,
			&task.Remark,
			&task.NodeDefKey,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (t *examineNode) ListByFlowID(ctx context.Context, flowID string) ([]model.Task, error) {
	rows, err := t.db.QueryContext(ctx,
		"select id, task_id,flow_id,user_id,substitute,created_by,examine_type,`result`,node_result,created_at,updated_at,app_id,form_table_id,form_data_id,form_ref,urge_times,remark,node_def_key from examine_node  WHERE flow_id=? order by created_at desc",
		flowID,
	)
	if err != nil {
		return nil, err
	}
	tasks := make([]model.Task, 0)
	for rows.Next() {
		task := model.Task{}
		err := rows.Scan(
			&task.ID,
			&task.TaskID,
			&task.FlowID,
			&task.UserID,
			&task.Substitute,
			&task.CreatedBy,
			&task.ExamineType,
			&task.Result,
			&task.NodeResult,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.AppID,
			&task.FormTableID,
			&task.FormDataID,
			&task.FormRef,
			&task.UrgeTimes,
			&task.Remark,
			&task.NodeDefKey,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
func (t *examineNode) GetByUserIDAndTaskID(ctx context.Context, userID, taskID string) (*model.Task, error) {
	row := t.db.QueryRowContext(ctx,
		"select id, task_id,flow_id,user_id,substitute,created_by,examine_type,`result`,node_result,created_at,updated_at,app_id,form_table_id,form_data_id,form_ref,urge_times,remark,node_def_key from examine_node  WHERE task_id = ? and user_id=?",
		taskID, userID,
	)
	task := model.Task{}
	err := row.Scan(
		&task.ID,
		&task.TaskID,
		&task.FlowID,
		&task.UserID,
		&task.Substitute,
		&task.CreatedBy,
		&task.ExamineType,
		&task.Result,
		&task.NodeResult,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.AppID,
		&task.FormTableID,
		&task.FormDataID,
		&task.FormRef,
		&task.UrgeTimes,
		&task.Remark,
		&task.NodeDefKey,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &task, err
}

func (t *examineNode) UpdateUrgeTimes(ctx context.Context, data *model.Task) error {
	_, err := t.db.ExecContext(ctx, "UPDATE `examine_node` SET `urge_times` = ?,updated_at=? WHERE id = ?",
		data.UrgeTimes,
		data.UpdatedAt,
		data.ID,
	)
	if err != nil {
		return err
	}
	return nil
}

func (t *examineNode) GetByID(ctx context.Context, id string) (*model.Task, error) {
	row := t.db.QueryRowContext(ctx,
		"select id, task_id,flow_id,user_id,substitute,created_by,examine_type,`result`,node_result,created_at,updated_at,app_id,form_table_id,form_data_id,form_ref,urge_times,remark,node_def_key from examine_node  WHERE id = ?",
		id,
	)
	task := model.Task{}
	err := row.Scan(
		&task.ID,
		&task.TaskID,
		&task.FlowID,
		&task.UserID,
		&task.Substitute,
		&task.CreatedBy,
		&task.ExamineType,
		&task.Result,
		&task.NodeResult,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.AppID,
		&task.FormTableID,
		&task.FormDataID,
		&task.FormRef,
		&task.UrgeTimes,
		&task.Remark,
		&task.NodeDefKey,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &task, err
}

func (t *examineNode) GetByUserID(ctx context.Context, userID, nodeResult string, page, limit int) ([]model.Task, int, error) {
	startIndex := (page - 1) * limit
	rows, err := t.db.QueryContext(ctx,
		"select id, task_id,flow_id,user_id,substitute,created_by,examine_type,`result`,node_result,created_at,updated_at,app_id,form_table_id,form_data_id,form_ref,urge_times,remark,node_def_key from examine_node  WHERE (user_id = ? or substitute=?) and node_result=? limit ?,?",
		userID,
		userID,
		nodeResult,
		startIndex,
		limit,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	var num int = 0
	countRow := t.db.QueryRowContext(ctx,
		"select count(id) as total from examine_node  WHERE (user_id = ? or substitute=?) and node_result=?",
		userID,
		userID,
		nodeResult,
	)
	err = countRow.Scan(&num)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	tasks := make([]model.Task, 0)
	for rows.Next() {
		task := model.Task{}
		err := rows.Scan(
			&task.ID,
			&task.TaskID,
			&task.FlowID,
			&task.UserID,
			&task.Substitute,
			&task.CreatedBy,
			&task.ExamineType,
			&task.Result,
			&task.NodeResult,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.AppID,
			&task.FormTableID,
			&task.FormDataID,
			&task.FormRef,
			&task.UrgeTimes,
			&task.Remark,
			&task.NodeDefKey,
		)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, num, err
}

func (t *examineNode) GetByCreated(ctx context.Context, userID string, page, limit int) ([]model.Task, int, error) {
	startIndex := (page - 1) * limit
	rows, err := t.db.QueryContext(ctx,
		"select id, task_id,flow_id,user_id,substitute,created_by,examine_type,`result`,node_result,created_at,updated_at,app_id,form_table_id,form_data_id,form_ref,urge_times,remark,node_def_key from examine_node  WHERE created_by = ? limit ?,?",
		userID,
		startIndex,
		limit,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	var num int = 0
	countRow := t.db.QueryRowContext(ctx,
		"select count(id) as total from examine_node  WHERE created_by = ?",
		userID,
	)
	err = countRow.Scan(&num)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	tasks := make([]model.Task, 0)
	for rows.Next() {
		task := model.Task{}
		err := rows.Scan(
			&task.ID,
			&task.TaskID,
			&task.FlowID,
			&task.UserID,
			&task.Substitute,
			&task.CreatedBy,
			&task.ExamineType,
			&task.Result,
			&task.NodeResult,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.AppID,
			&task.FormTableID,
			&task.FormDataID,
			&task.FormRef,
			&task.UrgeTimes,
			&task.Remark,
			&task.NodeDefKey,
		)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, num, err
}

//-------------------------
//
//func (t *examineNode) CountTimeOut(ctx context.Context, db *gorm.DB, userID, result, nodeResult string) (int64, error) {
//	if userID != "" {
//		db = db.Where("user_id=? or substitute=?", userID, userID, userID)
//	}
//	if result != "" {
//		db = db.Where("result=?", result)
//	}
//	if nodeResult != "" {
//		db = db.Where("node_result=?", nodeResult)
//	}
//	//todo 这里需要知道超时时间
//	var num int64
//	db.Table(t.TableName()).Count(&num)
//	return num, nil
//}
//
//func (t *examineNode) CountUrgeTimes(ctx context.Context, db *gorm.DB, userID, result, nodeResult string) (int64, error) {
//
//	if userID != "" {
//		db = db.Where("user_id=? or substitute=? ", userID, userID)
//	}
//	if result != "" {
//		db = db.Where("result=?", result)
//	}
//	if nodeResult != "" {
//		db = db.Where("node_result=?", nodeResult)
//	}
//	db = db.Where("urge_times > 1")
//	var num int64
//	db.Table(t.TableName()).Count(&num)
//
//	return num, nil
//}
//
//func (t *examineNode) GetByFlowInstanceID(ctx context.Context, db *gorm.DB, flowInstanceID string) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (t *examineNode) UpdateByFlowInstanceID(ctx context.Context, tx *gorm.DB, data *model.Task) error {
//	tx = tx.Table(t.TableName())
//	tx = tx.Where("flow_instance_id=?", data.FlowInstanceID)
//	err := tx.Updates(data).Error
//
//	if err != nil {
//		return err
//	}
//	return nil
//}
