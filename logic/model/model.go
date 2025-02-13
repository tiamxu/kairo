package model

import (
	"errors"
	"strings"
	"sync"

	"github.com/tiamxu/kit/sql"
	"gopkg.in/mgo.v2/bson"
)

var (
	postgresHandler   = sql.NewPreDB()
	mysqlHandler      = sql.NewPreDB()
	clickHouseHandler = sql.NewPreDB()
	dbMutex           sync.RWMutex
)

func Init(cfg *sql.Config) error {
	if cfg == nil {
		return errors.New("config cannot be nil")
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

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

	return err
}

func GetMysqlDB() *sql.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return mysqlHandler.DB
}

func GetPostgresDB() *sql.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return postgresHandler.DB
}

func GetClickhouseDB() *sql.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return clickHouseHandler.DB
}

func index(s string, sub ...string) int {
	minIndex := -1
	for _, ss := range sub {
		if idx := strings.Index(s, ss); idx != -1 && (minIndex == -1 || idx < minIndex) {
			minIndex = idx
		}
	}
	return minIndex
}

func insertZeroDeletedTsField(whereCond string) string {
	whereCond = strings.TrimSpace(whereCond)
	whereCond = strings.TrimRight(whereCond, ";")

	if strings.Contains(whereCond, "deleted_ts") {
		return whereCond
	}

	keywords := []string{"ORDER BY", "order by", "GROUP BY", "group by", "OFFSET", "offset", "LIMIT", "limit"}

	pos := index(whereCond, keywords...)

	if pos == -1 {
		return whereCond + " AND deleted_ts=0"
	}
	return whereCond[:pos] + " AND deleted_ts=0 " + whereCond[pos:]
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
