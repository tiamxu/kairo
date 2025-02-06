package logic

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tiamxu/kit/vectorstore"
	"github.com/tmc/langchaingo/schema"
)

type Service struct {
	db    *sql.DB
	store vectorstore.VectorStore
}

func (s *Service) StoreQA(ctx context.Context, question string, answer string) error {
	// 1. Store answer to MySQL
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO qa_pairs (question, answer) VALUES (?, ?)",
		question, answer)
	if err != nil {
		return fmt.Errorf("failed to store answer: %w", err)
	}

	// 2. Get generated ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// 3. Store question and ID to vector store
	doc := schema.Document{
		PageContent: question,
		Metadata: map[string]interface{}{
			"qa_id": id,
		},
	}

	err = s.store.AddDocuments(ctx, []schema.Document{doc})
	return err
}
