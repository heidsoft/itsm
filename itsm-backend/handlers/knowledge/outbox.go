package knowledge

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service"
)

// The Ent repository is the sole production implementation. Keeping this
// assertion local prevents mocks from accidentally becoming a second write
// path that cannot participate in the durable outbox.
func (s *Service) entRepository() (*EntRepository, error) {
	repo, ok := s.repo.(*EntRepository)
	if !ok {
		return nil, fmt.Errorf("knowledge vector outbox requires EntRepository")
	}
	return repo, nil
}

func (s *Service) createWithVectorOutbox(ctx context.Context, article *Article) (*Article, error) {
	if _, err := s.entRepository(); err != nil {
		return nil, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	created, err := (&EntRepository{client: tx.Client()}).Create(ctx, article)
	if err != nil {
		return nil, err
	}
	if created.IsPublished {
		if err := enqueueVectorSyncTx(ctx, tx, created, vectorIndexSync); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) updateWithVectorOutbox(ctx context.Context, article *Article) (*Article, error) {
	if _, err := s.entRepository(); err != nil {
		return nil, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	updated, err := (&EntRepository{client: tx.Client()}).Update(ctx, article)
	if err != nil {
		return nil, err
	}
	action := vectorIndexDelete
	if updated.IsPublished {
		action = vectorIndexSync
	}
	if err := enqueueVectorSyncTx(ctx, tx, updated, action); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) deleteWithVectorOutbox(ctx context.Context, id, tenantID int) error {
	if _, err := s.entRepository(); err != nil {
		return err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.KnowledgeArticle.Update().Where(
		knowledgearticle.IDEQ(id), knowledgearticle.TenantIDEQ(tenantID), knowledgearticle.DeletedAtIsNil(),
	).SetDeletedAt(time.Now()).Save(ctx); err != nil {
		return err
	}
	article := &Article{ID: id, TenantID: tenantID, UpdatedAt: time.Now()}
	if err := enqueueVectorSyncTx(ctx, tx, article, vectorIndexDelete); err != nil {
		return err
	}
	return tx.Commit()
}

// VectorIndexCommandHandler re-loads authoritative state inside the worker;
// it never trusts title/content or publication state from the command payload.
type VectorIndexCommandHandler struct {
	client *ent.Client
	rag    *service.RAGService
}

func NewVectorIndexCommandHandler(client *ent.Client, rag *service.RAGService) *VectorIndexCommandHandler {
	return &VectorIndexCommandHandler{client: client, rag: rag}
}

func (h *VectorIndexCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if h == nil || h.client == nil || h.rag == nil {
		return fmt.Errorf("knowledge vector sync is unavailable")
	}
	if cmd == nil || cmd.CommandType != commandbus.CommandSyncKnowledgeVector || cmd.AggregateType != "knowledge_article" {
		return fmt.Errorf("invalid knowledge vector command")
	}
	if cmd.Payload["action"] == vectorIndexDelete {
		return h.rag.RemoveArticle(ctx, cmd.TenantID, cmd.AggregateID)
	}
	article, err := h.client.KnowledgeArticle.Query().Where(
		knowledgearticle.IDEQ(cmd.AggregateID), knowledgearticle.TenantIDEQ(cmd.TenantID),
		knowledgearticle.DeletedAtIsNil(), knowledgearticle.IsPublished(true),
	).Only(ctx)
	if ent.IsNotFound(err) {
		return h.rag.RemoveArticle(ctx, cmd.TenantID, cmd.AggregateID)
	}
	if err != nil {
		return fmt.Errorf("load knowledge article: %w", err)
	}
	return h.rag.IndexArticle(ctx, cmd.TenantID, article.ID, article.Title, article.Content)
}
