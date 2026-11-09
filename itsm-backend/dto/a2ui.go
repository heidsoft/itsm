package dto

// A2UI 工单表单的消息结构。这些是「域协议」而非业务规则：handler 构造后直接
// json.Marshal 发给前端，service 侧生成初始表单时也复用同一套结构。
// 放在 dto 包可让 handlers/a2ui 的窄契约与消息构造都不再依赖共享层。

// A2UIDataModelUpdate 是 dataModelUpdate 消息信封。
type A2UIDataModelUpdate struct {
	DataModelUpdate DataModelUpdate `json:"dataModelUpdate"`
}

// DataModelUpdate 描述一次表单数据模型的局部更新。
type DataModelUpdate struct {
	SurfaceID string    `json:"surfaceId"`
	Path      string    `json:"path"`
	Contents  []Content `json:"contents"`
}

// Content 是 dataModelUpdate 中的单个键值更新。
type Content struct {
	Key         string `json:"key"`
	ValueString string `json:"valueString,omitempty"`
	ValueNumber *int   `json:"valueNumber,omitempty"`
	ValueBool   *bool  `json:"valueBoolean,omitempty"`
}

// A2UIDeleteSurface 是 deleteSurface 消息信封。
type A2UIDeleteSurface struct {
	DeleteSurface DeleteSurface `json:"deleteSurface"`
}

// DeleteSurface 描述要销毁的表单 surface。
type DeleteSurface struct {
	SurfaceID string `json:"surfaceId"`
}
