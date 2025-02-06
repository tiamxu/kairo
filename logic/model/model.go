package model

import (
	"errors"
	"strings"

	"github.com/tiamxu/kit/sql"
	"gopkg.in/mgo.v2/bson"
)

var postgresHandler = sql.NewPreDB()
var mysqlHandler = sql.NewPreDB()
var clickHouseHandler = sql.NewPreDB()

func Init(cfg *sql.Config) error {
	var err error
	switch strings.ToLower(cfg.Driver) {
	case "mysql":
		err = mysqlHandler.Init(cfg)
	case "postgres":
		err = postgresHandler.Init(cfg)
	case "clickhouse":
		err = clickHouseHandler.Init(cfg)
	default:
		return errors.New("unknown driver")
	}

	if err != nil {
		return err
	}

	return nil
}

func GetMysqlDB() *sql.DB {
	return mysqlHandler.DB
}

func GetPostgresDB() *sql.DB {
	return postgresHandler.DB
}

func GetClickhouseDB() *sql.DB {
	return clickHouseHandler.DB
}
func index(s string, sub ...string) int {
	var i, ii = -1, -1
	for _, ss := range sub {
		ii = strings.Index(s, ss)
		if ii != -1 && (ii < i || i == -1) {
			i = ii
		}
	}
	return i
}
func insertZeroDeletedTsField(whereCond string) string {
	whereCond = strings.TrimSpace(whereCond)
	whereCond = strings.TrimRight(whereCond, ";")
	i := index(
		whereCond,
		"deleted_ts",
		" deleted_ts",
	)
	if i != -1 {
		return whereCond
	}
	i = index(
		whereCond,
		"ORDER BY", "order by",
		"GROUP BY", "group by",
		"OFFSET", "offset",
		"LIMIT", "limit",
	)
	if i == -1 {
		return whereCond + " AND deleted_ts=0"
	}
	return whereCond[:i] + " AND deleted_ts=0 " + whereCond[i:]
}

func insertZeroDeletedTsM(m bson.M) bson.M {
	if _, found := m["deleted_ts"]; found {
		return m
	}
	m["deleted_ts"] = 0
	return m
}

type QaPairs struct {
	Id        int64  `key:"pri" json:"id,omitempty"`
	Question  string `json:"question,omitempty"`
	Answer    string `json:"answer,omitempty"`
	UpdatedTs int64  `json:"updated_ts,omitempty"`
	CreatedTs int64  `json:"created_ts,omitempty"`
	DeletedTs int64  `json:"deleted_ts,omitempty"`
}
