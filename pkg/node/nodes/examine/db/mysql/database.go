package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Mysql struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"db"`
	Log      bool   `yaml:"log"`
}

func NewDB(conf *Mysql) (*sql.DB, error) {
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s", conf.User, conf.Password, conf.Host, conf.Database)
	return sql.Open("mysql", connStr)
}
