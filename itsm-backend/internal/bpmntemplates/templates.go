// Package bpmntemplates 持有随二进制发布的内置 BPMN 模板，以及以模板集合为
// 唯一权威的两个派生口径（可部署 key 集合、内容摘要）。
//
// 模板本体与 embed FS 必须在同一个包内：Go 的 //go:embed 不能跨父目录引用，
// 这也是模板此前只能压在 service/ 下的原因。归位到 internal 之后，service
// 不再拥有模板文件，也不允许维护第二份模板清单。
package bpmntemplates

import "embed"

//go:embed bpmn/*.bpmn
var embedded embed.FS

// FS 返回内置模板的只读文件系统。
//
// 路径布局刻意保持为 "bpmn/<filename>.bpmn"，与迁移前 service 包的
// bpmnTemplates 完全一致：BPMNTemplateService 的 ReadFile / WalkDir 调用面
// 零改动，模板 key 集合与内容摘要也不会因为搬迁而漂移。
func FS() embed.FS { return embedded }
