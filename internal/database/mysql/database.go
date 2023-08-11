package mysql

import (
	"database/sql"
	"fmt"

	"git.yunify.com/quanxiang/workflow/internal/common"
	_ "github.com/go-sql-driver/mysql"
)

func NewDB(conf *common.Mysql) (*sql.DB, error) {

	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s", conf.User, conf.Password, conf.Host, conf.Database)
	return sql.Open("mysql", connStr)
}
