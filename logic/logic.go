package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/tiamxu/kit/llm"
	"github.com/tiamxu/kit/log"
	"github.com/tiamxu/kit/vectorstore"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
)

type LLMService struct {
	llm         llms.Model
	embedder    embeddings.Embedder
	vectorStore vectorstore.VectorStore
}

//	func NewLLMService() *LLMService {
//		return &LLMService{}
//	}
func NewLLMService(ctx context.Context, cfg *llm.Config, vectorStoreCfg vectorstore.VectorStoreConfig) (*LLMService, error) {
	service := &LLMService{}
	if err := service.Initialize(context.Background(), cfg, vectorStoreCfg); err != nil {
		return nil, err
	}
	return service, nil
}
func (s *LLMService) Initialize(ctx context.Context, cfg *llm.Config, vectorStoreCfg vectorstore.VectorStoreConfig) error {
	start := time.Now()
	defer func() {
		log.Printf("Model initialization completed in %v", time.Since(start))
	}()

	if err := s.setupLLMAndEmbedder(cfg); err != nil {
		return fmt.Errorf("model initialization failed: %w", err)
	}

	if err := s.setupVectorStore(ctx, vectorStoreCfg); err != nil {
		return fmt.Errorf("vector store initialization failed: %w", err)
	}

	return nil
}

func (s *LLMService) setupLLMAndEmbedder(cfg *llm.Config) error {
	llm, embedder, err := initializeLLMAndEmbedder(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM and embedder: %w", err)
	}

	s.llm = llm
	s.embedder = embedder
	return nil
}

func (s *LLMService) setupVectorStore(ctx context.Context, cfg vectorstore.VectorStoreConfig) error {
	vectorStore, err := initializeVectorStore(ctx, cfg, s.embedder)
	if err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}

	s.vectorStore = vectorStore
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
