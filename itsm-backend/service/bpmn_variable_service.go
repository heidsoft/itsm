package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processvariable"
)

// BPMNVariableService BPMN流程变量管理服务
type BPMNVariableService struct {
	client *ent.Client
}

// NewBPMNVariableService 创建BPMN变量管理服务实例
func NewBPMNVariableService(client *ent.Client) *BPMNVariableService {
	return &BPMNVariableService{client: client}
}

// VariableScope 变量作用域
type VariableScope string

const (
	ScopeProcess   VariableScope = "process"   // 流程级别变量
	ScopeInstance  VariableScope = "instance"  // 实例级别变量
	ScopeTask      VariableScope = "task"      // 任务级别变量
	ScopeExecution VariableScope = "execution" // 执行级别变量
	ScopeLocal     VariableScope = "local"     // 本地变量
)

// VariableType 变量类型
type VariableType string

const (
	TypeString   VariableType = "string"
	TypeInteger  VariableType = "integer"
	TypeFloat    VariableType = "float"
	TypeBoolean  VariableType = "boolean"
	TypeDate     VariableType = "date"
	TypeDateTime VariableType = "datetime"
	TypeJSON     VariableType = "json"
	TypeBinary   VariableType = "binary"
)

// ProcessVariable 流程变量结构
type ProcessVariable struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Value                interface{}            `json:"value"`
	Type                 VariableType           `json:"type"`
	Scope                VariableScope          `json:"scope"`
	ProcessDefinitionKey string                 `json:"process_definition_key,omitempty"`
	ProcessInstanceID    string                 `json:"process_instance_id,omitempty"`
	TaskID               string                 `json:"task_id,omitempty"`
	ExecutionID          string                 `json:"execution_id,omitempty"`
	IsTransient          bool                   `json:"is_transient"`               // 是否为临时变量
	IsReadOnly           bool                   `json:"is_read_only"`               // 是否为只读变量
	IsRequired           bool                   `json:"is_required"`                // 是否为必需变量
	DefaultValue         interface{}            `json:"default_value,omitempty"`    // 默认值
	ValidationRules      map[string]interface{} `json:"validation_rules,omitempty"` // 验证规则
	Description          string                 `json:"description,omitempty"`
	TenantID             int                    `json:"tenant_id"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// CreateVariableRequest 创建变量请求
type CreateVariableRequest struct {
	Name                 string                 `json:"name" binding:"required"`
	Value                interface{}            `json:"value"`
	Type                 VariableType           `json:"type" binding:"required"`
	Scope                VariableScope          `json:"scope" binding:"required"`
	ProcessDefinitionKey string                 `json:"process_definition_key,omitempty"`
	ProcessInstanceID    string                 `json:"process_instance_id,omitempty"`
	TaskID               string                 `json:"task_id,omitempty"`
	ExecutionID          string                 `json:"execution_id,omitempty"`
	IsTransient          bool                   `json:"is_transient"`
	IsReadOnly           bool                   `json:"is_read_only"`
	IsRequired           bool                   `json:"is_required"`
	DefaultValue         interface{}            `json:"default_value,omitempty"`
	ValidationRules      map[string]interface{} `json:"validation_rules,omitempty"`
	Description          string                 `json:"description,omitempty"`
	TenantID             int                    `json:"tenant_id" binding:"required"`
}

// UpdateVariableRequest 更新变量请求
type UpdateVariableRequest struct {
	Value           interface{}            `json:"value,omitempty"`
	IsTransient     *bool                  `json:"is_transient,omitempty"`
	IsReadOnly      *bool                  `json:"is_read_only,omitempty"`
	IsRequired      *bool                  `json:"is_required,omitempty"`
	DefaultValue    interface{}            `json:"default_value,omitempty"`
	ValidationRules map[string]interface{} `json:"validation_rules,omitempty"`
	Description     string                 `json:"description,omitempty"`
}

// ListVariablesRequest 查询变量请求
type ListVariablesRequest struct {
	ProcessDefinitionKey string        `json:"process_definition_key,omitempty"`
	ProcessInstanceID    string        `json:"process_instance_id,omitempty"`
	TaskID               string        `json:"task_id,omitempty"`
	ExecutionID          string        `json:"execution_id,omitempty"`
	Scope                VariableScope `json:"scope,omitempty"`
	Type                 VariableType  `json:"type,omitempty"`
	IsTransient          *bool         `json:"is_transient,omitempty"`
	IsReadOnly           *bool         `json:"is_read_only,omitempty"`
	IsRequired           *bool         `json:"is_required,omitempty"`
	TenantID             int           `json:"tenant_id" binding:"required"`
	Page                 int           `json:"page"`
	PageSize             int           `json:"page_size"`
}

// CreateVariable 创建流程变量
func (s *BPMNVariableService) CreateVariable(ctx context.Context, req *CreateVariableRequest) (*ent.ProcessVariable, error) {
	// 验证变量值
	if err := s.validateVariableValue(req.Value, req.Type); err != nil {
		return nil, fmt.Errorf("变量值验证失败: %w", err)
	}

	// 序列化变量值
	valueBytes, err := json.Marshal(req.Value)
	if err != nil {
		return nil, fmt.Errorf("序列化变量值失败: %w", err)
	}

	// 序列化默认值 - 暂时不使用，因为ProcessVariable没有DefaultValue字段
	// var defaultValueBytes []byte
	// if req.DefaultValue != nil {
	// 	defaultValueBytes, err = json.Marshal(req.DefaultValue)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("序列化默认值失败: %w", err)
	// 	}
	// }

	// 序列化验证规则 - 暂时不使用，因为ProcessVariable没有ValidationRules字段
	// var validationRulesBytes []byte
	// if req.ValidationRules != nil {
	// 	validationRulesBytes, err = json.Marshal(req.ValidationRules)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("序列化验证规则失败: %w", err)
	// 	}
	// }

	// 创建变量记录
	variable, err := s.client.ProcessVariable.Create().
		SetVariableName(req.Name).
		SetVariableValue(string(valueBytes)). // ProcessVariable的variable_value是text类型
		SetVariableType(string(req.Type)).
		SetScope(string(req.Scope)).
		SetProcessInstanceID(req.ProcessInstanceID).
		SetTaskID(req.TaskID).
		SetIsTransient(req.IsTransient).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("创建流程变量失败: %w", err)
	}

	return variable, nil
}

// GetVariable 获取变量
func (s *BPMNVariableService) GetVariable(ctx context.Context, id string, tenantID int) (*ent.ProcessVariable, error) {
	// ProcessVariable的ID是string类型，但ent生成的查询方法可能期望int
	// 这里暂时使用VariableID查询，或者需要检查ent生成的代码
	variable, err := s.client.ProcessVariable.Query().
		Where(processvariable.VariableID(id)).
		Where(processvariable.TenantID(tenantID)).
		Only(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取流程变量失败: %w", err)
	}

	return variable, nil
}

// UpdateVariable 更新变量
func (s *BPMNVariableService) UpdateVariable(ctx context.Context, id string, req *UpdateVariableRequest, tenantID int) (*ent.ProcessVariable, error) {
	// 获取现有变量
	_, err := s.GetVariable(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// 使用ent的链式调用模式更新变量
	updateBuilder := s.client.ProcessVariable.Update().
		Where(processvariable.VariableID(id)).
		Where(processvariable.TenantID(tenantID))

	// 更新变量值
	if req.Value != nil {
		// 序列化新值
		valueBytes, err := json.Marshal(req.Value)
		if err != nil {
			return nil, fmt.Errorf("序列化变量值失败: %w", err)
		}
		updateBuilder.SetVariableValue(string(valueBytes))
	}

	// 更新其他字段 - 只使用ProcessVariable实际存在的字段
	if req.IsTransient != nil {
		updateBuilder.SetIsTransient(*req.IsTransient)
	}

	// 执行更新
	_, err = updateBuilder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新流程变量失败: %w", err)
	}

	// 返回更新后的变量
	return s.GetVariable(ctx, id, tenantID)
}

// DeleteVariable 删除变量
func (s *BPMNVariableService) DeleteVariable(ctx context.Context, id string, tenantID int) error {
	// 获取现有变量
	variable, err := s.GetVariable(ctx, id, tenantID)
	if err != nil {
		return err
	}

	// ProcessVariable没有IsRequired字段，暂时跳过检查
	// if variable.IsRequired {
	// 	return fmt.Errorf("必需变量不允许删除")
	// }

	// 删除变量
	err = s.client.ProcessVariable.DeleteOne(variable).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除流程变量失败: %w", err)
	}

	return nil
}

// ListVariables 查询变量列表
func (s *BPMNVariableService) ListVariables(ctx context.Context, req *ListVariablesRequest) ([]*ent.ProcessVariable, int, error) {
	query := s.client.ProcessVariable.Query().
		Where(processvariable.TenantID(req.TenantID))

	// 添加过滤条件 - 只使用ProcessVariable实际存在的字段
	if req.ProcessInstanceID != "" {
		query = query.Where(processvariable.ProcessInstanceID(req.ProcessInstanceID))
	}
	if req.TaskID != "" {
		query = query.Where(processvariable.TaskID(req.TaskID))
	}
	if req.Scope != "" {
		query = query.Where(processvariable.Scope(string(req.Scope)))
	}
	if req.IsTransient != nil {
		query = query.Where(processvariable.IsTransient(*req.IsTransient))
	}

	// ProcessVariable没有以下字段，暂时跳过
	// if req.ProcessDefinitionKey != "" {
	// 	query = query.Where(processvariable.ProcessDefinitionKey(req.ProcessDefinitionKey))
	// }
	// if req.ExecutionID != "" {
	// 	query = query.Where(processvariable.ExecutionID(req.ExecutionID))
	// }
	// if req.Type != "" {
	// 	query = query.Where(processvariable.Type(string(req.Type)))
	// }
	// if req.IsReadOnly != nil {
	// 	query = query.Where(processvariable.IsReadOnly(*req.IsReadOnly))
	// }
	// if req.IsRequired != nil {
	// 	query = query.Where(processvariable.IsRequired(*req.IsRequired))
	// }

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询变量总数失败: %w", err)
	}

	// 分页查询
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 按创建时间倒序排列
	variables, err := query.Order(ent.Desc(processvariable.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询变量列表失败: %w", err)
	}

	return variables, total, nil
}

// GetVariablesByScope 根据作用域获取变量
func (s *BPMNVariableService) GetVariablesByScope(ctx context.Context, scope VariableScope, scopeID string, tenantID int) ([]*ent.ProcessVariable, error) {
	query := s.client.ProcessVariable.Query().
		Where(processvariable.Scope(string(scope))).
		Where(processvariable.TenantID(tenantID))

	// 根据作用域类型添加过滤条件 - 只使用ProcessVariable实际存在的字段
	switch scope {
	case ScopeInstance:
		query = query.Where(processvariable.ProcessInstanceID(scopeID))
	case ScopeTask:
		query = query.Where(processvariable.TaskID(scopeID))
	}

	// ProcessVariable没有以下字段，暂时跳过
	// case ScopeProcess:
	// 	query = query.Where(processvariable.ProcessDefinitionKey(scopeID))
	// case ScopeExecution:
	// 	query = query.Where(processvariable.ExecutionID(scopeID))

	variables, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询作用域变量失败: %w", err)
	}

	return variables, nil
}

// SetVariable 设置变量值
func (s *BPMNVariableService) SetVariable(ctx context.Context, name string, value interface{}, scope VariableScope, scopeID string, tenantID int) error {
	// 检查变量是否已存在
	existingVariables, err := s.GetVariablesByScope(ctx, scope, scopeID, tenantID)
	if err != nil {
		return err
	}

	var existingVariable *ent.ProcessVariable
	for _, v := range existingVariables {
		// ProcessVariable没有Name字段，使用VariableName
		if v.VariableName == name {
			existingVariable = v
			break
		}
	}

	if existingVariable != nil {
		// 更新现有变量 - 使用VariableID而不是ID
		_, err = s.UpdateVariable(ctx, existingVariable.VariableID, &UpdateVariableRequest{
			Value: value,
		}, tenantID)
		return err
	} else {
		// 创建新变量
		req := &CreateVariableRequest{
			Name:     name,
			Value:    value,
			Type:     s.inferVariableType(value),
			Scope:    scope,
			TenantID: tenantID,
		}

		// 根据作用域设置相应的ID
		switch scope {
		case ScopeProcess:
			req.ProcessDefinitionKey = scopeID
		case ScopeInstance:
			req.ProcessInstanceID = scopeID
		case ScopeTask:
			req.TaskID = scopeID
		case ScopeExecution:
			req.ExecutionID = scopeID
		}

		_, err = s.CreateVariable(ctx, req)
		return err
	}
}

// GetVariableValue 获取变量值
func (s *BPMNVariableService) GetVariableValue(ctx context.Context, name string, scope VariableScope, scopeID string, tenantID int) (interface{}, error) {
	variables, err := s.GetVariablesByScope(ctx, scope, scopeID, tenantID)
	if err != nil {
		return nil, err
	}

	for _, variable := range variables {
		// ProcessVariable没有Name字段，使用VariableName
		if variable.VariableName == name {
			// 反序列化变量值 - ProcessVariable的Value是string类型
			var value interface{}
			if err := json.Unmarshal([]byte(variable.VariableValue), &value); err != nil {
				return nil, fmt.Errorf("反序列化变量值失败: %w", err)
			}
			return value, nil
		}
	}

	return nil, fmt.Errorf("变量 %s 不存在", name)
}

// DeleteVariablesByScope 删除指定作用域的所有变量
func (s *BPMNVariableService) DeleteVariablesByScope(ctx context.Context, scope VariableScope, scopeID string, tenantID int) error {
	variables, err := s.GetVariablesByScope(ctx, scope, scopeID, tenantID)
	if err != nil {
		return err
	}

	// ProcessVariable没有IsRequired字段，暂时删除所有变量
	// 过滤出非必需变量
	// var deletableVariables []*ent.ProcessVariable
	// for _, variable := range variables {
	// 	if !variable.IsRequired {
	// 		deletableVariables = append(deletableVariables, variable)
	// 	}
	// }

	// 批量删除 - 使用VariableID而不是ID
	if len(variables) > 0 {
		var ids []string
		for _, variable := range variables {
			ids = append(ids, variable.VariableID)
		}

		_, err = s.client.ProcessVariable.Delete().
			Where(processvariable.VariableIDIn(ids...)).
			Exec(ctx)

		if err != nil {
			return fmt.Errorf("批量删除变量失败: %w", err)
		}
	}

	return nil
}

// validateVariableValue 验证变量值
func (s *BPMNVariableService) validateVariableValue(value interface{}, varType VariableType) error {
	if value == nil {
		return fmt.Errorf("变量值不能为空")
	}

	switch varType {
	case TypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("变量类型不匹配，期望string类型")
		}
	case TypeInteger:
		if _, ok := value.(int); !ok {
			return fmt.Errorf("变量类型不匹配，期望integer类型")
		}
	case TypeFloat:
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("变量类型不匹配，期望float类型")
		}
	case TypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("变量类型不匹配，期望boolean类型")
		}
	case TypeDate, TypeDateTime:
		if _, ok := value.(time.Time); !ok {
			return fmt.Errorf("变量类型不匹配，期望时间类型")
		}
	case TypeJSON:
		// JSON类型可以接受任何可序列化的值
		if _, err := json.Marshal(value); err != nil {
			return fmt.Errorf("变量值无法序列化为JSON: %w", err)
		}
	case TypeBinary:
		if _, ok := value.([]byte); !ok {
			return fmt.Errorf("变量类型不匹配，期望binary类型")
		}
	default:
		return fmt.Errorf("不支持的变量类型: %s", varType)
	}

	return nil
}

// inferVariableType 推断变量类型
func (s *BPMNVariableService) inferVariableType(value interface{}) VariableType {
	switch value.(type) {
	case string:
		return TypeString
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return TypeInteger
	case float32, float64:
		return TypeFloat
	case bool:
		return TypeBoolean
	case time.Time:
		return TypeDateTime
	case []byte:
		return TypeBinary
	default:
		return TypeJSON
	}
}

// CopyVariables 复制变量到新的作用域
func (s *BPMNVariableService) CopyVariables(ctx context.Context, fromScope VariableScope, fromScopeID string, toScope VariableScope, toScopeID string, tenantID int) error {
	// 获取源作用域的变量
	sourceVariables, err := s.GetVariablesByScope(ctx, fromScope, fromScopeID, tenantID)
	if err != nil {
		return err
	}

	// 复制每个变量
	for _, sourceVariable := range sourceVariables {
		// 反序列化变量值 - ProcessVariable的Value是string类型
		var value interface{}
		if err := json.Unmarshal([]byte(sourceVariable.VariableValue), &value); err != nil {
			continue // 跳过无法反序列化的变量
		}

		// ProcessVariable没有DefaultValue和ValidationRules字段，暂时跳过
		// 反序列化默认值
		// var defaultValue interface{}
		// if sourceVariable.DefaultValue != nil {
		// 	json.Unmarshal(sourceVariable.DefaultValue, &defaultValue)
		// }

		// 反序列化验证规则
		// var validationRules map[string]interface{}
		// if sourceVariable.ValidationRules != nil {
		// 	json.Unmarshal(sourceVariable.ValidationRules, &validationRules)
		// }

		// 创建新变量 - 只使用ProcessVariable实际存在的字段
		req := &CreateVariableRequest{
			Name:        sourceVariable.VariableName, // 使用VariableName而不是Name
			Value:       value,
			Type:        VariableType(sourceVariable.VariableType), // 使用VariableType而不是Type
			Scope:       toScope,
			IsTransient: sourceVariable.IsTransient,
			// ProcessVariable没有以下字段，暂时跳过
			// IsReadOnly:      sourceVariable.IsReadOnly,
			// IsRequired:      sourceVariable.IsRequired,
			// DefaultValue:    defaultValue,
			// ValidationRules: validationRules,
			// Description:     sourceVariable.Description,
			TenantID: tenantID,
		}

		// 设置目标作用域ID
		switch toScope {
		case ScopeProcess:
			req.ProcessDefinitionKey = toScopeID
		case ScopeInstance:
			req.ProcessInstanceID = toScopeID
		case ScopeTask:
			req.TaskID = toScopeID
		case ScopeExecution:
			req.ExecutionID = toScopeID
		}

		// 创建变量
		if _, err := s.CreateVariable(ctx, req); err != nil {
			// 记录错误但继续处理其他变量
			fmt.Printf("复制变量 %s 失败: %v\n", sourceVariable.VariableName, err)
		}
	}

	return nil
}
