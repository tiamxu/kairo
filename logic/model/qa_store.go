package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/tiamxu/kit/sql"
)

var (
	QaPairsTableName = "\"qa_pairs\""
)

func (*QaPairs) TableName() string {
	return QaPairsTableName
}

func GetQaPairsDB() *sql.DB {
	return mysqlHandler.DB
}

// StoreAnswer 存储问答到数据库
func StoreAnswer(ctx context.Context, question string, answer string) (int64, error) {
	result, err := GetQaPairsDB().ExecContext(ctx,
		"INSERT INTO qa_pairs (question, answer) VALUES (?, ?)",
		question, answer)

	if err != nil {
		return 0, fmt.Errorf("failed to store answer: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return id, nil
}

// GetAnswersByIDs 根据ID列表获取提示词
func GetAnswersByIDs(ctx context.Context, ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = fmt.Sprintf("%d", id)
	}
	idList := strings.Join(idStrs, ",")

	var answers []string
	querySql := fmt.Sprintf("SELECT answer FROM qa_pairs WHERE id IN (%s)", idList)
	err := GetQaPairsDB().SelectContext(ctx, &answers, querySql)
	if err != nil {
		return nil, fmt.Errorf("failed to query answers: %w", err)
	}
	return answers, nil
}

// GetStoredQuestions 获取所有存储的问题
func GetStoredQuestions(ctx context.Context) ([]string, error) {
	questions := []string{}
	query := "SELECT question FROM qa_pairs ORDER BY created_at DESC"
	if err := GetQaPairsDB().SelectContext(ctx, &questions, query); err != nil {
		return nil, fmt.Errorf("failed to query questions: %w", err)
	}
	return questions, nil
}

