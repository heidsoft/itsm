package service

import (
	"context"
	"strings"
	"time"

	"itsm-backend/ent"
	ia "itsm-backend/ent/incident"
	ka "itsm-backend/ent/knowledgearticle"

	"go.uber.org/zap"
)

// Embedder defines interface to turn text into vector []float32
type Embedder interface {
	Embed(text string) ([]float32, error)
}

// EmbeddingPipeline scans knowledge articles and closed incidents to produce vectors.
type EmbeddingPipeline struct {
	client   *ent.Client
	embedder Embedder
	logger   *zap.SugaredLogger
	vectors  *VectorStore
}

func NewEmbeddingPipeline(client *ent.Client, e Embedder, logger *zap.SugaredLogger, vectors *VectorStore) *EmbeddingPipeline {
	return &EmbeddingPipeline{client: client, embedder: e, logger: logger, vectors: vectors}
}

// RunOnce embeds latest N articles (simplified). In production, use job/queue.
func (p *EmbeddingPipeline) RunOnce(ctx context.Context, tenantID int, limit int) error {
	if limit <= 0 {
		limit = 20
	}
	arts, err := p.client.KnowledgeArticle.Query().Where(ka.TenantIDEQ(tenantID)).Limit(limit).All(ctx)
	if err != nil {
		return err
	}
	for _, a := range arts {
		text := a.Title + "\n" + strings.TrimSpace(a.Content)
		vec, err := p.embedder.Embed(text)
		if err != nil {
			continue
		}
		if p.vectors != nil {
			_ = p.vectors.Upsert(ctx, tenantID, "kb", a.ID, vec, a.Content, "kb:"+a.Category)
		}
		if p.logger != nil {
			p.logger.Infow("Embedded KB", "id", a.ID, "tenant_id", tenantID, "ts", time.Now().Unix())
		}
	}
	// also embed latest incidents (title + description) for similarity search
	incs, err := p.client.Incident.Query().Where(ia.TenantIDEQ(tenantID)).Order(ent.Desc(ia.FieldCreatedAt)).Limit(limit).All(ctx)
	if err == nil {
		for _, it := range incs {
			text := strings.TrimSpace(it.Title + "\n" + it.Description)
			if text == "" {
				continue
			}
			vec, err := p.embedder.Embed(text)
			if err != nil {
				continue
			}
			if p.vectors != nil {
				_ = p.vectors.Upsert(ctx, tenantID, "incident", it.ID, vec, it.Description, "incident:"+it.IncidentNumber)
			}
			if p.logger != nil {
				p.logger.Infow("Embedded Incident", "id", it.ID, "tenant_id", tenantID, "ts", time.Now().Unix())
			}
		}
	}
	return nil
}
