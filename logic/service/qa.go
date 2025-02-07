package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/tiamxu/kit/llm"
	"github.com/tiamxu/kit/log"

	"github.com/tiamxu/kit/vectorstore"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type Service struct {
	llm   llms.Model
	db    *sql.DB
	store vectorstore.VectorStore
}

func NewService(ctx context.Context, cfg *llm.Config, db *sql.DB, vectorStoreCfg vectorstore.VectorStoreConfig) (*Service, error) {
	service := &Service{
		db: db,
	}
	if err := service.Initialize(ctx, cfg, vectorStoreCfg); err != nil {
		return nil, err
	}
	return service, nil
}
func (s *Service) Initialize(ctx context.Context, cfg *llm.Config, vectorStoreCfg vectorstore.VectorStoreConfig) error {
	start := time.Now()
	defer func() {
		log.Printf("Service initialization completed in %v", time.Since(start))
	}()

	if err := s.setupLLMAndEmbedder(cfg); err != nil {
		return fmt.Errorf("model initialization failed: %w", err)
	}

	if err := s.setupVectorStore(ctx, vectorStoreCfg); err != nil {
		return fmt.Errorf("vector store initialization failed: %w", err)
	}

	return nil
}
func (s *Service) setupLLMAndEmbedder(cfg *llm.Config) error {
	llm, _, err := initializeLLMAndEmbedder(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM and embedder: %w", err)
	}

	s.llm = llm
	// 注意：这里不需要直接使用 embedder，因为我们只需要它来初始化 vector store
	return nil
}
func (s *Service) setupVectorStore(ctx context.Context, cfg vectorstore.VectorStoreConfig) error {
	vectorStore, err := initializeVectorStore(ctx, cfg, nil) // 如果不直接使用 embedder，则传入 nil
	if err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}

	s.store = vectorStore
	return nil
}
func initializeLLMAndEmbedder(cfg *llm.Config) (llms.Model, embeddings.Embedder, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("llm config is nil")
	}

	llm, embedder, err := llm.NewModels(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize Models (type: %s): %w", cfg.Type, err)
	}
	return llm, embedder, nil
}
func initializeVectorStore(ctx context.Context, cfg vectorstore.VectorStoreConfig, embedder embeddings.Embedder) (vectorstore.VectorStore, error) {
	if cfg.Type == "" {
		return nil, fmt.Errorf("vector store type is empty")
	}

	var store vectorstore.VectorStore
	var err error

	switch cfg.Type {
	case "milvus":
		store = vectorstore.NewMilvusStore(&cfg.Milvus, embedder)
	case "qdrant":
		store = vectorstore.NewQdrantStore(&cfg.Qdrant, embedder)
	default:
		return nil, fmt.Errorf("unsupported vector store type: %s", cfg.Type)
	}

	if err = store.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize %s store: %w", cfg.Type, err)
	}

	return store, nil
}
func (s *Service) retrieveDocuments(ctx context.Context, query string, topK int) ([]schema.Document, error) {
	if s.store == nil {
		return nil, fmt.Errorf("store is not initialized")
	}
	// 使用 Search 方法替代 SimilaritySearch
	docs, err := s.store.Search(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}
	return docs, nil
}
func (s *Service) RetrieveAnswer(ctx context.Context, query string, topK int) ([]string, error) {
	// 1. Vector search for similar questions
	docs, err := s.retrieveDocuments(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	// 2. Extract IDs
	var ids []int64
	for _, doc := range docs {
		if id, ok := doc.Metadata["qa_id"]; ok {
			switch v := id.(type) {
			case int64:
				ids = append(ids, v)
			case float64:
				ids = append(ids, int64(v))
			default:
				return nil, fmt.Errorf("unexpected type for qa_id: %T", id)
			}
		}
	}

	// 3. Query answers from MySQL
	if len(ids) == 0 {
		return nil, nil
	}

	// Convert ids to comma-separated string
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = fmt.Sprintf("%d", id)
	}
	idList := strings.Join(idStrs, ",")

	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf("SELECT answer FROM qa_pairs WHERE id IN (%s)", idList))
	if err != nil {
		return nil, fmt.Errorf("failed to query answers: %w", err)
	}
	defer rows.Close()

	// 4. Collect answers
	var answers []string
	for rows.Next() {
		var answer string
		if err := rows.Scan(&answer); err != nil {
			return nil, fmt.Errorf("failed to scan answer: %w", err)
		}
		answers = append(answers, answer)
	}

	return answers, nil
}

func (s *Service) generateFinalResponse(ctx context.Context, query string, answers []string) (string, error) {
	// Create prompt with question and answers
	prompt := fmt.Sprintf("根据以下问题和相关答案，生成一个完整的回答：\n\n问题：%s\n\n相关答案：\n%s",
		query, strings.Join(answers, "\n\n"))

	// Generate response using LLM
	res, err := s.llm.GenerateContent(ctx, []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(prompt)},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate final response: %w", err)
	}

	return res.Choices[0].Content, nil
}
func (s *Service) QueryWithRetrieve(ctx context.Context, query string, topK int) (string, error) {
	start := time.Now()
	defer func() {
		log.Printf("QueryWithRetrieve completed in %v", time.Since(start))
	}()

	// Retrieve answers from database
	answers, err := s.RetrieveAnswer(ctx, query, topK)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve answers: %w", err)
	}

	// If no answers found, return default response
	if len(answers) == 0 {
		return "没有找到相关答案", nil
	}

	// Generate final response using LLM
	return s.generateFinalResponse(ctx, query, answers)
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

func (s *Service) GetStoredQuestions(ctx context.Context) ([]string, error) {
	fmt.Println("###:")

	rows, err := s.db.QueryContext(ctx, "SELECT question FROM qa_pairs ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to query questions: %w", err)
	}
	defer rows.Close()

	var questions []string
	for rows.Next() {
		var question string
		if err := rows.Scan(&question); err != nil {
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}
		questions = append(questions, question)
	}
	return questions, nil
}
