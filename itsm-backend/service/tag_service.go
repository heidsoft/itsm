package service

import (
	"context"
	"fmt"
	
	"itsm-backend/ent"
	"itsm-backend/ent/tag"
)

type TagService struct {
	client *ent.Client
}

func NewTagService(client *ent.Client) *TagService {
	return &TagService{client: client}
}

// CreateTag 创建通用标签
func (s *TagService) CreateTag(ctx context.Context, name, code, description, color string, tenantID int) (*ent.Tag, error) {
	return s.client.Tag.Create().
		SetName(name).
		SetCode(code).
		SetDescription(description).
		SetColor(color).
		SetTenantID(tenantID).
		Save(ctx)
}

// ListTags 获取标签列表
func (s *TagService) ListTags(ctx context.Context, tenantID int) ([]*ent.Tag, error) {
	return s.client.Tag.Query().
		Where(tag.TenantID(tenantID)).
		All(ctx)
}

// BindTagToEntity 为实体绑定标签
func (s *TagService) BindTagToEntity(ctx context.Context, tagID int, entityType string, entityID int) error {
	tagClient := s.client.Tag
	update := tagClient.UpdateOneID(tagID)

	switch entityType {
	case "project":
		update.AddProjectIDs(entityID)
	case "application":
		update.AddApplicationIDs(entityID)
	case "microservice":
		update.AddMicroserviceIDs(entityID)
	case "department":
		update.AddDepartmentIDs(entityID)
	case "team":
		update.AddTeamIDs(entityID)
	default:
		return fmt.Errorf("unsupported entity type: %s", entityType)
	}
	
	return update.Exec(ctx)
}

// UpdateTag 更新标签
func (s *TagService) UpdateTag(ctx context.Context, id int, name, code, description, color *string, tenantID int) (*ent.Tag, error) {
	// 检查标签是否存在且属于当前租户
	_, err := s.client.Tag.Query().
		Where(
			tag.IDEQ(id),
			tag.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}

	update := s.client.Tag.UpdateOneID(id)
	if name != nil {
		update = update.SetName(*name)
	}
	if code != nil {
		update = update.SetCode(*code)
	}
	if description != nil {
		update = update.SetDescription(*description)
	}
	if color != nil {
		update = update.SetColor(*color)
	}

	return update.Save(ctx)
}

// DeleteTag 删除标签
func (s *TagService) DeleteTag(ctx context.Context, id int, tenantID int) error {
	// 检查标签是否存在且属于当前租户
	exists, err := s.client.Tag.Query().
		Where(
			tag.IDEQ(id),
			tag.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("标签不存在: id=%d", id)
	}

	// 删除标签
	return s.client.Tag.DeleteOneID(id).Exec(ctx)
}
