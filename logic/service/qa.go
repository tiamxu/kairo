package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tiamxu/kit/llm"
	"github.com/tiamxu/kit/log"
	"github.com/tiamxu/kit/sql"

	"github.com/tiamxu/kairo/logic/model"
	"github.com/tiamxu/kit/vectorstore"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type Service struct {
	llm      llms.Model
	embedder embeddings.Embedder
	store    vectorstore.VectorStore
}

func NewLLMService(ctx context.Context, cfg *llm.Config, vectorStoreCfg *vectorstore.VectorStoreConfig, dbConfig *sql.Config) (*Service, error) {
	service := &Service{}
	if err := service.Initialize(ctx, cfg, vectorStoreCfg, dbConfig); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) Initialize(ctx context.Context, cfg *llm.Config, vectorStoreCfg *vectorstore.VectorStoreConfig, dbConfig *sql.Config) error {
	start := time.Now()
	defer func() {
		log.Printf("Model initialization completed in %v", time.Since(start))
	}()

	// 初始化 LLM 和 Embedder
	llm, embedder, err := s.initializeLLMAndEmbedder(cfg)
	if err != nil {
		return fmt.Errorf("model initialization failed: %w", err)
	}

	// 初始化向量存储
	store, err := s.initializeVectorStore(ctx, vectorStoreCfg, embedder)
	if err != nil {
		return fmt.Errorf("vector store initialization failed: %w", err)
	}

	// 初始化数据库
	// if err := model.InitDB(dbConfig); err != nil {
	// 	return fmt.Errorf("database initialization failed: %w", err)
	// }

	s.llm = llm
	s.embedder = embedder
	s.store = store
	return nil
}

func (s *Service) initializeLLMAndEmbedder(cfg *llm.Config) (llms.Model, embeddings.Embedder, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("llm config is nil")
	}

	llm, embedder, err := llm.NewModels(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize Models (type: %s): %w", cfg.Type, err)
	}
	return llm, embedder, nil
}

func (s *Service) initializeVectorStore(ctx context.Context, cfg *vectorstore.VectorStoreConfig, embedder embeddings.Embedder) (vectorstore.VectorStore, error) {
	if cfg.Type == "" {
		return nil, fmt.Errorf("vector store type is empty")
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder is nil")
	}
	var store vectorstore.VectorStore
	var err error

	switch cfg.Type {
	case "milvus":
		store = vectorstore.NewMilvusStore(&cfg.Milvus, embedder)
	case "qdrant":
		if err := cfg.Qdrant.Validate(); err != nil {
			return nil, fmt.Errorf("invalid qdrant configuration: %w", err)
		}
		store = vectorstore.NewQdrantStore(&cfg.Qdrant, embedder)
	default:
		return nil, fmt.Errorf("unsupported vector store type: %s", cfg.Type)
	}
	if store == nil {
		return nil, fmt.Errorf("store creation failed: store is nil after creation")
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
	docs, err := s.retrieveDocuments(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

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

	return model.GetAnswersByIDs(ctx, ids)
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
	// Store QA pair in database
	id, err := model.StoreAnswer(ctx, question, answer)
	if err != nil {
		return fmt.Errorf("failed to store QA pair: %w", err)
	}

	// Store question in vector store
	doc := schema.Document{
		PageContent: question,
		Metadata: map[string]interface{}{
			"qa_id": id,
		},
	}

	return s.store.AddDocuments(ctx, []schema.Document{doc})
}

func (s *Service) GetStoredQuestions(ctx context.Context) ([]string, error) {
	return model.GetStoredQuestions(ctx)
}
