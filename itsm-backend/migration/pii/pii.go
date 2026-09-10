// Package pii 提供「列级 PII 标签」与「脱敏策略」基础设施：
//
//  1. 在 ent schema 的 field 上用 .Annotations(pii.New("email")) 等标注 PII 列；
//  2. PolicyFromDescriptors 从 ent runtime descriptor 提取 (table, column) → Strategy 映射；
//  3. MaskExpr / MaskedCopySQL 产出"脱敏副本"SQL，给开发 / 预发环境安全灌入；
//  4. ValidateNotNullCheck / ParseMigrationDDL 用 NOT NULL / DEFAULT 校核 + DDL 解析
//     帮迁移 PR 把"会破坏老读路径"的新列拦在合入前。
//
// 单一源：策略只有 Strategy 枚举里的几类；新增需要先在 ent schema 加注解再扩策略。
package pii

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"entgo.io/ent/schema"
)

// Strategy 列脱敏策略。名字与外部 plugin / SQL 函数保持一致，便于运维记忆。
type Strategy string

const (
	StrategyEmail     Strategy = "email"     // hash + salt；格式不可逆但同 email 仍相同
	StrategyPhone     Strategy = "phone"     // hash + salt；保留 E.164 长度
	StrategyIDCard    Strategy = "id_card"   // hash + salt；中国身份证等
	StrategyName      Strategy = "name"      // 张* / 李**；格式保留
	StrategyAddress   Strategy = "address"   // 部分脱敏；区 / 路保留，门牌号 *
	StrategyAPIKey    Strategy = "api_key"   // NULL；mask 列标记为 REDACTED
	StrategyFreeText  Strategy = "free_text" // 整段替换为固定占位符，禁止外发
)

// AllStrategies 是允许列表；任何不在表内的注解在 Policy 提取阶段会被忽略并记日志。
func AllStrategies() []Strategy {
	return []Strategy{StrategyEmail, StrategyPhone, StrategyIDCard, StrategyName, StrategyAddress, StrategyAPIKey, StrategyFreeText}
}

func IsValidStrategy(s string) bool {
	for _, v := range AllStrategies() {
		if string(v) == s {
			return true
		}
	}
	return false
}

// saltKey 在 hash 类策略上混入 HMAC 盐；同一原文在同 salt 下输出稳定 hash，
// 但不能跨 salt 比对，避免"反向查表"。
var saltKey = []byte("itsm-pii-mask-v1")

// MaskExpr 给一个 (table, column) 对返回脱敏后的 SQL 表达式，喂给 SELECT 列表使用：
//
//   - hash 类策略：encode(hmac_sha256(saltKey, value)) 前 16 字节
//   - name / address：用正则只保留首字 + 替换中间字符为 *
//   - api_key / free_text：直接返回固定占位符
//
// 接受 raw string 类型；非 string 列（int / bool / timestamp）调用方应跳过。
func MaskExpr(table, column string, strategy Strategy) (string, error) {
	switch strategy {
	case StrategyEmail, StrategyPhone, StrategyIDCard:
		// 使用 substring + 编码保证不同长度邮箱产生不同 hash，避免完全相同列值碰撞
		// Postgres: substring(encode(hmac(...), 'hex'), 1, 16)
		// SQLite:   substr(hex(...) ...), 1, 16)
		// 给出兼容两端的子查询：encode 在 pg 上有，hex 是 SQLite 名。
		// 这里给 PG 风格；SQLite fallback 在 MaskedCopySQL 里处理。
		col := quoteIdent(table) + "." + quoteIdent(column)
		return fmt.Sprintf("substring(encode(hmac(%s, %q::bytea), 'hex'), 1, 16)", col, string(saltKey)), nil
	case StrategyName:
		// 张*：保留首字 + 替换其余中文字符为 *
		return fmt.Sprintf("regexp_replace(%s.%s, '([\\p{L}])([\\p{L}\\p{N}]+)', '\\1*', 'g')",
			quoteIdent(table), quoteIdent(column)), nil
	case StrategyAddress:
		return fmt.Sprintf("regexp_replace(%s.%s, '([0-9])', '*', 'g')",
			quoteIdent(table), quoteIdent(column)), nil
	case StrategyAPIKey, StrategyFreeText:
		return fmt.Sprintf("'%s'", redactedPlaceholder(strategy)), nil
	default:
		return "", fmt.Errorf("pii: unknown strategy %q", strategy)
	}
}

// redactedPlaceholder 给定策略返回 REDACTED 占位符；带策略前缀便于日志/审计快速定位。
func redactedPlaceholder(s Strategy) string {
	return fmt.Sprintf("REDACTED:%s", s)
}

// quoteIdent 简单包裹 ANSI SQL 引号；只接受 [a-zA-Z0-9_]+，避免 SQL 注入与方言差异。
func quoteIdent(name string) string {
	if !identRE.MatchString(name) {
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(name, "\"", "\"\""))
	}
	return fmt.Sprintf("\"%s\"", name)
}

var identRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// FieldDescriptor 是 ent 生成的 FieldDescriptor 抽象。
// 仅暴露工具链需要的最小字段，避免耦合 ent 内部 API。
type FieldDescriptor struct {
	Table  string // 物理表名（snake_case）
	Name   string // 字段名
	GoName string // Go 字段名，用于校验报错信息
	Type   string // "string" / "int" / "time" / "json" / "bool" / ...
	Annotations []schema.Annotation
}

// Policy (table, column) → Strategy 的扁平映射；同时记录每个 table 的所有列供
// MaskedCopySQL 生成完整 SELECT 列表使用。
type Policy struct {
	// TableColumns[table] = []column；按 ent schema 声明顺序
	TableColumns map[string][]string
	// Strategies[table][column] = Strategy
	Strategies map[string]map[string]Strategy
}

// Empty 判断是否没有任何标注。
func (p Policy) Empty() bool {
	return len(p.Strategies) == 0
}

// Lookup 给出 (table, column) → Strategy，未标注返回 ("", false)。
func (p Policy) Lookup(table, column string) (Strategy, bool) {
	if mp, ok := p.Strategies[table]; ok {
		s, ok := mp[column]
		return s, ok
	}
	return "", false
}

// PolicyFromDescriptors 从 ent 运行时 FieldDescriptor 列表里抽取 PII 注解。
// 入参通常来自 ent.Client 的内部 descriptor，但为不绑死实现，调用方自行构造 FieldDescriptor。
// 任何无效策略会被跳过并返回 ErrInvalidStrategy，调用方应 fail-closed 拒绝发布。
func PolicyFromDescriptors(descriptors []FieldDescriptor) (Policy, error) {
	p := Policy{TableColumns: map[string][]string{}, Strategies: map[string]map[string]Strategy{}}
	for _, d := range descriptors {
		if _, ok := p.TableColumns[d.Table]; !ok {
			p.TableColumns[d.Table] = nil
			p.Strategies[d.Table] = map[string]Strategy{}
		}
		p.TableColumns[d.Table] = append(p.TableColumns[d.Table], d.Name)
		for _, a := range d.Annotations {
			ann, ok := a.(*Annotation)
			if !ok {
				continue
			}
			if !IsValidStrategy(string(ann.Strategy)) {
				return Policy{}, fmt.Errorf("%w: table=%s column=%s got=%s", ErrInvalidStrategy, d.Table, d.Name, ann.Strategy)
			}
			p.Strategies[d.Table][d.Name] = Strategy(ann.Strategy)
		}
	}
	return p, nil
}

// Annotation 实现 entgo.schema.Annotation；name 用于 ent template 反射。
// 我们不走 ent template（避免重型 codegen），直接挂在 entgo.Annotation 上，
// PolicyFromDescriptors 通过 type assertion 拿回 *Annotation。
type Annotation struct {
	Strategy Strategy
}

// Name 满足 schema.Annotation 接口。
func (a *Annotation) Name() string { return "PII" }

// New 构造一个 PII 注解，挂在 field.String("email").Annotations(pii.New("email"))。
func New(strategy Strategy) schema.Annotation {
	return &Annotation{Strategy: strategy}
}

var (
	ErrInvalidStrategy = errors.New("pii: invalid strategy")
	ErrEmptyMaskedCopy = errors.New("pii: no PII columns, no masked copy needed")
)

// MaskedCopySQL 生成"create table mask_<table> as select ..."风格的脱敏副本 SQL。
// 一次只生成一张表的副本；多张表请循环调用。
//
// 该 SQL 假设：
//   - 表已经存在且迁移已应用；
//   - 数据库为 PostgreSQL（encode/bytea/hex）；
//   - 仅对 String 列应用 hash / 部分脱敏，对其它类型列原样拷贝。
//
// 对于 SQLite 测试环境，工具链会调 SQLiteVariant 把 encode/bytea 替换为 hex/lengthB。
func MaskedCopySQL(table string, columns []string, p Policy, saltOverride string) (string, error) {
	if p.Empty() {
		return "", ErrEmptyMaskedCopy
	}
	cols := columns
	if len(cols) == 0 {
		cols = p.TableColumns[table]
	}
	if len(cols) == 0 {
		return "", fmt.Errorf("pii: no columns registered for table %s", table)
	}
	salt := saltKey
	if saltOverride != "" {
		salt = []byte(saltOverride)
	}
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		strategy, ok := p.Lookup(table, c)
		if !ok {
			out = append(out, fmt.Sprintf("%s.%s", quoteIdent(table), quoteIdent(c)))
			continue
		}
		expr, err := maskExprForStrategy(table, c, strategy, salt)
		if err != nil {
			return "", err
		}
		out = append(out, expr+" AS "+quoteIdent(c))
	}
	create := fmt.Sprintf("CREATE TABLE IF NOT EXISTS mask_%s AS SELECT %s FROM %s WHERE FALSE",
		table, strings.Join(out, ", "), quoteIdent(table))
	return create, nil
}

// maskExprForStrategy 是 MaskExpr 的内部版本，接受外部 salt。
func maskExprForStrategy(table, column string, strategy Strategy, salt []byte) (string, error) {
	switch strategy {
	case StrategyEmail, StrategyPhone, StrategyIDCard:
		col := quoteIdent(table) + "." + quoteIdent(column)
		return fmt.Sprintf("substring(encode(hmac(%s, %s), 'hex'), 1, 16)",
			col, fmt.Sprintf("decode(md5(%s::text), 'hex')", col)), nil
	case StrategyName:
		return fmt.Sprintf("regexp_replace(%s.%s, '([\\p{L}])([\\p{L}\\p{N}]+)', '\\1*', 'g')",
			quoteIdent(table), quoteIdent(column)), nil
	case StrategyAddress:
		return fmt.Sprintf("regexp_replace(%s.%s, '([0-9])', '*', 'g')",
			quoteIdent(table), quoteIdent(column)), nil
	case StrategyAPIKey, StrategyFreeText:
		return fmt.Sprintf("'%s'", redactedPlaceholder(strategy)), nil
	default:
		return "", fmt.Errorf("pii: unknown strategy %q", strategy)
	}
}

// HashValue 暴露在 cmd / 测试工具里，对一条原始值返回稳定 hash；便于单元测试。
func HashValue(value string, salt []byte) string {
	if len(salt) == 0 {
		salt = saltKey
	}
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))[:16]
}