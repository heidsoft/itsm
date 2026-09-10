package pii

import (
	"entgo.io/ent/entc/load"
	"entgo.io/ent/schema"
)

// ExtractFromLoadedSchema 把 entc 加载的 schema (load.Schema) 转为扁平
// FieldDescriptor 列表，供 PolicyFromDescriptors 使用。
//
// 调用方通常：
//   schema, err := load.Load("./ent/schema")
//   descriptors, err := pii.ExtractFromLoadedSchema(schema)
//   policy, err := pii.PolicyFromDescriptors(descriptors)
//
// 这里不直接 walk ent.Schema 接口，避免把 entc/gen 这种重型依赖倒灌到 pii 包里。
//
// entc/load 把 annotations 反序列化为 map[string]interface{} 而不是直接还原
// 类型，所以这里同时识别两种形态：*Annotation（直接类型）和
// {"Strategy": "email"}（map 形态）。
func ExtractFromLoadedSchema(schemas []*load.Schema) ([]FieldDescriptor, error) {
	out := make([]FieldDescriptor, 0)
	for _, s := range schemas {
		table := snake(s.Name)
		for _, f := range s.Fields {
			strategies := extractStrategies(f.Annotations)
			anns := make([]schema.Annotation, 0, len(strategies))
			for _, strategy := range strategies {
				if IsValidStrategy(strategy) {
					anns = append(anns, &Annotation{Strategy: Strategy(strategy)})
				}
			}
			out = append(out, FieldDescriptor{
				Table:       table,
				Name:        snake(f.Name),
				GoName:      f.Name,
				Type:        f.Info.Ident,
				Annotations: anns,
			})
		}
	}
	return out, nil
}

func extractStrategies(annotations map[string]any) []string {
	out := make([]string, 0, len(annotations))
	for name, v := range annotations {
		// entc/load 反序列化时,key = annotation.Name() (即 "PII"),value = 字段 map;
		// 我们只关心 PII 注解,因此过滤 name。
		if name != "PII" {
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		strategy, ok := m["Strategy"].(string)
		if !ok {
			continue
		}
		out = append(out, strategy)
	}
	return out
}

// AnnotationAlias 是 PII Annotation 的可序列化影子，避免 pii 包反向依赖 ent schema 类型。
type AnnotationAlias struct {
	Name     string
	Strategy string
}

// snake 把 CamelCase 转 snake_case，跟 ent 默认 StorageKey 行为一致。
func snake(s string) string {
	out := make([]rune, 0, len(s)+4)
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_')
		}
		if r >= 'A' && r <= 'Z' {
			out = append(out, r+'a'-'A')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}