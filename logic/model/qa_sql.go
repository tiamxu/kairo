package model

import "github.com/tiamxu/kit/sql"

var (
	QaPairsTableName = "\"qa_pairs\""
)

func (*QaPairs) TableName() string {
	return QaPairsTableName
}

func GetQaPairsDB() *sql.DB {
	return mysqlHandler.DB
}

func SelectWithQaPairs() {

}
