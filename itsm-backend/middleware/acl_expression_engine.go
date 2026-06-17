package middleware

import (
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// =============================================================================
// ACL 表达式引擎 - 递归下降解析器
//
// 支持：
//   - 比较运算: ==, !=, >, <, >=, <=
//   - 逻辑运算: &&, ||, !
//   - 变量引用: ctx.user_id, ctx.tenant_id, ctx.role, ctx.resource_id, ctx.resource_type, ctx.resource.owner_id
//   - 字面量: 整数、字符串（双引号）
//
// 安全策略：
//   - 空字符串 → return true（无 ACL 即放行）
//   - 解析失败/执行异常 → return false（安全侧兜底）+ Error 日志
// =============================================================================

// TokenType 词法单元类型
type TokenType int

const (
	TokenIdentifier TokenType = iota // 标识符（ctx.user_id, ctx.role 等）
	TokenNumber                      // 数字字面量
	TokenString                      // 字符串字面量
	TokenOperator                    // 比较运算符 (==, !=, >, <, >=, <=)
	TokenLogicOp                     // 逻辑运算符 (&&, ||)
	TokenNot                         // 逻辑非 (!)
	TokenLParen                      // 左括号
	TokenRParen                      // 右括号
	TokenDot                         // 点号（属性访问）
	TokenEOF                         // 结束
)

// Token 词法单元
type Token struct {
	Type  TokenType
	Value string
}

// Lexer 词法分析器
type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

// NewLexer 创建词法分析器
func NewLexer(input string) *Lexer {
	return &Lexer{input: input, pos: 0}
}

// Tokenize 执行词法分析
func (l *Lexer) Tokenize() ([]Token, error) {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]

		// 跳过空白
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			l.pos++
			continue
		}

		// 双字符运算符
		if l.pos+1 < len(l.input) {
			twoChar := l.input[l.pos : l.pos+2]
			switch twoChar {
			case "==":
				l.tokens = append(l.tokens, Token{TokenOperator, "=="})
				l.pos += 2
				continue
			case "!=":
				l.tokens = append(l.tokens, Token{TokenOperator, "!="})
				l.pos += 2
				continue
			case ">=":
				l.tokens = append(l.tokens, Token{TokenOperator, ">="})
				l.pos += 2
				continue
			case "<=":
				l.tokens = append(l.tokens, Token{TokenOperator, "<="})
				l.pos += 2
				continue
			case "&&":
				l.tokens = append(l.tokens, Token{TokenLogicOp, "&&"})
				l.pos += 2
				continue
			case "||":
				l.tokens = append(l.tokens, Token{TokenLogicOp, "||"})
				l.pos += 2
				continue
			}
		}

		// 单字符运算符和符号
		switch ch {
		case '>':
			l.tokens = append(l.tokens, Token{TokenOperator, ">"})
			l.pos++
		case '<':
			l.tokens = append(l.tokens, Token{TokenOperator, "<"})
			l.pos++
		case '!':
			l.tokens = append(l.tokens, Token{TokenNot, "!"})
			l.pos++
		case '(':
			l.tokens = append(l.tokens, Token{TokenLParen, "("})
			l.pos++
		case ')':
			l.tokens = append(l.tokens, Token{TokenRParen, ")"})
			l.pos++
		case '.':
			l.tokens = append(l.tokens, Token{TokenDot, "."})
			l.pos++
		case '"':
			// 字符串字面量
			str, err := l.readString()
			if err != nil {
				return nil, err
			}
			l.tokens = append(l.tokens, Token{TokenString, str})
		default:
			if isDigit(ch) {
				// 数字
				num := l.readNumber()
				l.tokens = append(l.tokens, Token{TokenNumber, num})
			} else if isLetter(ch) || ch == '_' {
				// 标识符
				ident := l.readIdentifier()
				l.tokens = append(l.tokens, Token{TokenIdentifier, ident})
			} else {
				return nil, fmt.Errorf("非法字符: %c (位置 %d)", ch, l.pos)
			}
		}
	}

	l.tokens = append(l.tokens, Token{TokenEOF, ""})
	return l.tokens, nil
}

func (l *Lexer) readString() (string, error) {
	l.pos++ // 跳过开头的 "
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '"' {
			l.pos++ // 跳过结尾的 "
			return sb.String(), nil
		}
		if ch == '\\' && l.pos+1 < len(l.input) {
			// 转义字符
			l.pos++
			ch = l.input[l.pos]
		}
		sb.WriteByte(ch)
		l.pos++
	}
	return "", fmt.Errorf("字符串未闭合")
}

func (l *Lexer) readNumber() string {
	var sb strings.Builder
	for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
		sb.WriteByte(l.input[l.pos])
		l.pos++
	}
	return sb.String()
}

func (l *Lexer) readIdentifier() string {
	var sb strings.Builder
	for l.pos < len(l.input) && (isLetter(l.input[l.pos]) || isDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		sb.WriteByte(l.input[l.pos])
		l.pos++
	}
	return sb.String()
}

func isDigit(ch byte) bool  { return ch >= '0' && ch <= '9' }
func isLetter(ch byte) bool { return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') }

// ACLExpressionEngine ACL 表达式引擎
type ACLExpressionEngine struct{}

// NewACLExpressionEngine 创建 ACL 表达式引擎
func NewACLExpressionEngine() *ACLExpressionEngine {
	return &ACLExpressionEngine{}
}

// Evaluate 评估 ACL 表达式
// 变量映射：ctx.user_id, ctx.tenant_id, ctx.role, ctx.resource_id, ctx.resource_type, ctx.resource.owner_id 等
func (e *ACLExpressionEngine) Evaluate(expression string, variables map[string]interface{}) bool {
	if expression == "" {
		return true // 空 ACL 即放行
	}

	// 词法分析
	lexer := NewLexer(expression)
	tokens, err := lexer.Tokenize()
	if err != nil {
		zap.S().Errorw("ACL 表达式词法分析失败",
			"expression", expression,
			"error", err,
		)
		return false // 安全侧兜底
	}

	// 语法分析和执行
	parser := newACLParser(tokens, variables)
	result, err := parser.parseOr()
	if err != nil {
		zap.S().Errorw("ACL 表达式解析/执行失败",
			"expression", expression,
			"error", err,
		)
		return false // 安全侧兜底
	}

	return result
}

// aclParser 递归下降解析器
type aclParser struct {
	tokens    []Token
	pos       int
	variables map[string]interface{}
}

func newACLParser(tokens []Token, variables map[string]interface{}) *aclParser {
	return &aclParser{
		tokens:    tokens,
		pos:       0,
		variables: variables,
	}
}

// 当前 token
func (p *aclParser) current() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{TokenEOF, ""}
}

// 前进到下一个 token
func (p *aclParser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

// 期望特定 token 类型
func (p *aclParser) expect(t TokenType) (Token, error) {
	tok := p.advance()
	if tok.Type != t {
		return tok, fmt.Errorf("期望 token 类型 %d, 实际 %d (%s)", t, tok.Type, tok.Value)
	}
	return tok, nil
}

// parseOr: or_expr = and_expr ("||" and_expr)*
func (p *aclParser) parseOr() (bool, error) {
	left, err := p.parseAnd()
	if err != nil {
		return false, err
	}

	for p.current().Type == TokenLogicOp && p.current().Value == "||" {
		p.advance() // 消费 ||
		right, err := p.parseAnd()
		if err != nil {
			return false, err
		}
		left = left || right
	}

	return left, nil
}

// parseAnd: and_expr = not_expr ("&&" not_expr)*
func (p *aclParser) parseAnd() (bool, error) {
	left, err := p.parseNot()
	if err != nil {
		return false, err
	}

	for p.current().Type == TokenLogicOp && p.current().Value == "&&" {
		p.advance() // 消费 &&
		right, err := p.parseNot()
		if err != nil {
			return false, err
		}
		left = left && right
	}

	return left, nil
}

// parseNot: not_expr = "!" not_expr | primary
func (p *aclParser) parseNot() (bool, error) {
	if p.current().Type == TokenNot {
		p.advance() // 消费 !
		operand, err := p.parseNot()
		if err != nil {
			return false, err
		}
		return !operand, nil
	}
	return p.parseComparison()
}

// parseComparison: comparison = value (op value)?
func (p *aclParser) parseComparison() (bool, error) {
	left, err := p.parseValue()
	if err != nil {
		return false, err
	}

	// 如果当前是比较运算符
	if p.current().Type == TokenOperator {
		op := p.advance().Value
		right, err := p.parseValue()
		if err != nil {
			return false, err
		}
		return compareValues(left, right, op)
	}

	// 否则将值转为布尔值
	return toBool(left), nil
}

// parseValue: value = number | string | identifier | "(" or_expr ")"
func (p *aclParser) parseValue() (interface{}, error) {
	tok := p.current()

	switch tok.Type {
	case TokenNumber:
		p.advance()
		num, err := strconv.Atoi(tok.Value)
		if err != nil {
			return nil, fmt.Errorf("无效数字: %s", tok.Value)
		}
		return num, nil

	case TokenString:
		p.advance()
		return tok.Value, nil

	case TokenIdentifier:
		p.advance()
		// 构建变量路径：ctx.user_id, ctx.resource.owner_id 等
		varPath := tok.Value
		for p.current().Type == TokenDot {
			p.advance() // 消费 .
			next, err := p.expect(TokenIdentifier)
			if err != nil {
				return nil, err
			}
			varPath += "." + next.Value
		}
		// 查找变量值
		val, exists := p.variables[varPath]
		if !exists {
			// 变量不存在，返回 nil（比较时将返回 false）
			return nil, fmt.Errorf("变量未定义: %s", varPath)
		}
		return val, nil

	case TokenLParen:
		p.advance() // 消费 (
		result, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, fmt.Errorf("意外的 token: %s (类型 %d)", tok.Value, tok.Type)
	}
}

// compareValues 比较两个值
func compareValues(left, right interface{}, op string) (bool, error) {
	// 将两值统一为可比较的类型
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	// 尝试数值比较
	leftNum, leftIsNum := toFloat64(left)
	rightNum, rightIsNum := toFloat64(right)

	if leftIsNum && rightIsNum {
		switch op {
		case "==":
			return leftNum == rightNum, nil
		case "!=":
			return leftNum != rightNum, nil
		case ">":
			return leftNum > rightNum, nil
		case "<":
			return leftNum < rightNum, nil
		case ">=":
			return leftNum >= rightNum, nil
		case "<=":
			return leftNum <= rightNum, nil
		}
	}

	// 字符串比较
	switch op {
	case "==":
		return leftStr == rightStr, nil
	case "!=":
		return leftStr != rightStr, nil
	case ">":
		return leftStr > rightStr, nil
	case "<":
		return leftStr < rightStr, nil
	case ">=":
		return leftStr >= rightStr, nil
	case "<=":
		return leftStr <= rightStr, nil
	}

	return false, fmt.Errorf("不支持的运算符: %s", op)
}

// toFloat64 尝试将值转为 float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

// toBool 将值转为布尔值
func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val != "" && val != "0" && val != "false"
	case nil:
		return false
	default:
		return v != nil
	}
}
