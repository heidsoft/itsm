package pii

import (
	"strings"
	"testing"

	"entgo.io/ent/schema"

	"github.com/stretchr/testify/require"
)

func TestStrategyValidation(t *testing.T) {
	require.True(t, IsValidStrategy("email"))
	require.True(t, IsValidStrategy("phone"))
	require.True(t, IsValidStrategy("id_card"))
	require.False(t, IsValidStrategy("bogus"))
}

func TestPolicyFromDescriptorsValidatesStrategy(t *testing.T) {
	descriptors := []FieldDescriptor{
		{Table: "users", Name: "email", Annotations: []schema.Annotation{New(StrategyEmail)}},
		{Table: "users", Name: "phone", Annotations: []schema.Annotation{New(StrategyPhone)}},
		{Table: "users", Name: "name"}, // no annotation → OK
	}
	p, err := PolicyFromDescriptors(descriptors)
	require.NoError(t, err)
	require.Len(t, p.Strategies, 1)
	require.Equal(t, StrategyEmail, p.Strategies["users"]["email"])
	require.Equal(t, StrategyPhone, p.Strategies["users"]["phone"])
	_, ok := p.Lookup("users", "name")
	require.False(t, ok)
}

func TestPolicyFromDescriptorsRejectsInvalidStrategy(t *testing.T) {
	descriptors := []FieldDescriptor{
		{Table: "users", Name: "email", Annotations: []schema.Annotation{
			&Annotation{Strategy: "unknown"},
		}},
	}
	_, err := PolicyFromDescriptors(descriptors)
	require.ErrorIs(t, err, ErrInvalidStrategy)
}

func TestMaskedCopySQLGeneratesExpectedShape(t *testing.T) {
	p := Policy{
		TableColumns: map[string][]string{"users": {"id", "email", "phone", "name"}},
		Strategies: map[string]map[string]Strategy{
			"users": {"email": StrategyEmail, "phone": StrategyPhone, "name": StrategyName},
		},
	}
	sql, err := MaskedCopySQL("users", nil, p, "")
	require.NoError(t, err)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS mask_users")
	require.Contains(t, sql, "FROM \"users\"")
	require.Contains(t, sql, "substring(encode(hmac(")
	require.Contains(t, sql, "AS \"email\"")
	require.Contains(t, sql, "AS \"phone\"")
	require.Contains(t, sql, "regexp_replace(")
	require.Contains(t, sql, "WHERE FALSE")
}

func TestMaskedCopySQLSkipsUntaggedColumns(t *testing.T) {
	p := Policy{
		TableColumns: map[string][]string{"users": {"id", "email"}},
		Strategies:   map[string]map[string]Strategy{"users": {"email": StrategyEmail}},
	}
	sql, err := MaskedCopySQL("users", nil, p, "")
	require.NoError(t, err)
	require.Contains(t, sql, "\"users\".\"id\"") // plain column
	require.Contains(t, sql, "AS \"email\"")     // masked column
}

func TestMaskedCopySQLRejectsUnknownTable(t *testing.T) {
	p := Policy{TableColumns: map[string][]string{}, Strategies: map[string]map[string]Strategy{}}
	_, err := MaskedCopySQL("nope", nil, p, "")
	require.Error(t, err)
}

func TestMaskExprAPIReturnsPlaceholder(t *testing.T) {
	expr, err := MaskExpr("users", "api_key", StrategyAPIKey)
	require.NoError(t, err)
	require.Contains(t, expr, "REDACTED:api_key")
}

func TestHashValueDeterministicAndSalted(t *testing.T) {
	a := HashValue("alice@example.com", []byte("salt-1"))
	b := HashValue("alice@example.com", []byte("salt-1"))
	require.Equal(t, a, b, "same salt 必须给稳定 hash")
	require.Len(t, a, 16)
	c := HashValue("alice@example.com", []byte("salt-2"))
	require.NotEqual(t, a, c, "salt 变化必须改变 hash")
}

func TestQuoteIdentRejectsSQLMeta(t *testing.T) {
	require.Equal(t, `"users"`, quoteIdent("users"))
	require.Equal(t, `"a-b"`, quoteIdent("a-b"))
}

func TestMaskExprFailsOnUnknownStrategy(t *testing.T) {
	_, err := MaskExpr("u", "c", Strategy("nope"))
	require.Error(t, err)
}

func TestMaskedCopySQLContainsTableQuoting(t *testing.T) {
	p := Policy{
		TableColumns: map[string][]string{"x": {"c"}},
		Strategies:   map[string]map[string]Strategy{"x": {"c": StrategyAPIKey}},
	}
	sql, err := MaskedCopySQL("x", nil, p, "")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(sql, "CREATE TABLE IF NOT EXISTS mask_x"))
}