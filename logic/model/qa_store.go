package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	QaPairsTableName = "\"qa_pairs\""
)

func (*QaPairs) TableName() string {
	return QaPairsTableName
}

// StoreAnswer 存储问答到数据库
func StoreAnswer(ctx context.Context, question string, answer string) (int64, error) {
	db := GetMysqlDB()
	if db == nil {
		return 0, errors.New("数据库连接未初始化")
	}

	result, err := db.ExecContext(ctx,
		"INSERT INTO qa_pairs (question, answer) VALUES (?, ?)",
		question, answer)

	if err != nil {
		return 0, fmt.Errorf("存储答案失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取插入ID失败: %w", err)
	}

	return id, nil
}

// GetAnswersByIDs 根据ID列表获取提示词
func GetAnswersByIDs(ctx context.Context, ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	db := GetMysqlDB()
	if db == nil {
		return nil, errors.New("数据库连接未初始化")
	}

	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = fmt.Sprintf("%d", id)
	}
	idList := strings.Join(idStrs, ",")

	var answers []string
	querySql := fmt.Sprintf("SELECT answer FROM qa_pairs WHERE id IN (%s) AND deleted_ts = 0", idList)
	err := db.SelectContext(ctx, &answers, querySql)
	if err != nil {
		return nil, fmt.Errorf("查询答案失败: %w", err)
	}
	return answers, nil
}

// GetStoredQuestions 获取所有存储的问题
func GetStoredQuestions(ctx context.Context) ([]string, error) {
	db := GetMysqlDB()
	if db == nil {
		return nil, errors.New("数据库连接未初始化")
	}

	questions := []string{}
	query := `
        SELECT question
        FROM qa_pairs 
        WHERE deleted_ts = 0 
        ORDER BY created_ts	 DESC
    `

	if err := db.SelectContext(ctx, &questions, query); err != nil {
		return nil, fmt.Errorf("查询问题列表失败: %w", err)
	}
	return questions, nil
}

// DeleteQaPair 删除问答对
func DeleteQaPair(ctx context.Context, id int64) error {
	db := GetMysqlDB()
	if db == nil {
		return errors.New("数据库连接未初始化")
	}

	result, err := db.ExecContext(ctx,
		"UPDATE qa_pairs SET deleted_ts = ? WHERE id = ?",
		time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("删除问答对失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if affected == 0 {
		return errors.New("未找到要删除的记录")
	}

	return nil
}

// UpdateQaPair 更新问答对
func UpdateQaPair(ctx context.Context, id int64, question, answer string) error {
	db := GetMysqlDB()
	if db == nil {
		return errors.New("数据库连接未初始化")
	}

	result, err := db.ExecContext(ctx,
		"UPDATE qa_pairs SET question = ?, answer = ?, updated_ts = ? WHERE id = ? AND deleted_ts = 0",
		question, answer, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("更新问答对失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if affected == 0 {
		return errors.New("未找到要更新的记录")
	}

	return nil
}
