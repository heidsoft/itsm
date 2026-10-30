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

// vectorIndexBlocked reports a wired vector backend whose embedder cannot
// generate vectors. Enqueueing a sync in that state only burns retries until the
// command dead-letters, so it must not be treated as pending work.
func (s *Service) vectorIndexBlocked() bool {
	return s.rag != nil && s.rag.HasVectorBackend() && !s.rag.VectorIndexReady()
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
		if s.vectorIndexBlocked() {
			s.logger.Warnw("knowledge article published without vector indexing: embedder is not configured",
				"article_id", created.ID, "tenant_id", created.TenantID)
		} else if err := enqueueVectorSyncTx(ctx, tx, created, vectorIndexSync); err != nil {
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
	// 用 updateTx 而非 EntRepository.Update：后者会自开一个独立事务，
	// 外层回滚时它已提交，指针与索引任务就不再同生共死。
	updated, err := updateTx(ctx, tx.Client(), article)
	if err != nil {
		return nil, err
	}
	// 默认入删除：未发布，以及「已发布但 embedding 不可用」都必须清掉旧索引，
	// 只有能力就绪的发布才入同步。
	action := vectorIndexDelete
	if updated.IsPublished {
		if s.vectorIndexBlocked() {
			s.logger.Warnw("knowledge article published while embedder is not configured; invalidating vector index",
				"article_id", updated.ID, "tenant_id", updated.TenantID)
		} else {
			action = vectorIndexSync
		}
	}
	if err := enqueueVectorSyncTx(ctx, tx, updated, action); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

// setPublishedWithVectorOutbox 发布 / 撤回，并把「线上版本」指针与索引任务
// 放进同一事务。
//
// 同事务的理由与 updateWithVectorOutbox 一致但更关键：一旦指针与索引分属
// 两个事务，就可能出现「指针说线上是 v3，索引里还是 v2 的正文」——而撤回
// 场景下反过来更危险：指针清了、索引删除没入库，RAG 仍会引用已撤下的内容。
func (s *Service) setPublishedWithVectorOutbox(ctx context.Context, id, tenantID int, published bool) (*Article, error) {
	if _, err := s.entRepository(); err != nil {
		return nil, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var updated *Article
	if published {
		updated, err = publishWithPointerTx(ctx, tx.Client(), id, tenantID)
	} else {
		updated, err = unpublishWithPointerTx(ctx, tx.Client(), id, tenantID)
	}
	if err != nil {
		return nil, err
	}

	action := vectorIndexDelete
	if updated.IsPublished {
		if s.vectorIndexBlocked() {
			s.logger.Warnw("knowledge article published while embedder is not configured; invalidating vector index",
				"article_id", updated.ID, "tenant_id", updated.TenantID)
		} else {
			action = vectorIndexSync
		}
	}
	if err := enqueueVectorSyncTx(ctx, tx, updated, action); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) restoreWithVectorOutbox(ctx context.Context, articleID, version, tenantID, userID int) (*Article, error) {
	if _, err := s.entRepository(); err != nil {
		return nil, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	restored, err := (&EntRepository{client: tx.Client()}).RestoreVersion(ctx, articleID, version, tenantID, userID)
	if err != nil {
		return nil, err
	}
	// 恢复后文章回到 draft 且撤销发布，旧索引必须同事务清理。
	if err := enqueueVectorSyncTx(ctx, tx, restored, vectorIndexDelete); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return restored, nil
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
