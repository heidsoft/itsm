package vector

import (
	"strings"
	"testing"
)

// 过滤键必须落到 metadata JSONB 文本投影：pgvector 表仅四列，
// 裸列名 WHERE（tenantID 等）因列不存在必然报错，检索静默降级。
func TestSearchFilterClauses_MetadataProjection(t *testing.T) {
	where, args, err := searchFilterClauses(map[string]interface{}{"tenantID": 2, "objectType": "kb"}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(where) != 2 || len(args) != 2 {
		t.Fatalf("expected 2 clauses/args, got %d/%d", len(where), len(args))
	}

	seenKey := map[string]bool{}
	argSet := map[string]bool{}
	for _, clause := range where {
		switch {
		case strings.HasPrefix(clause, "metadata->>'tenantID' = $"):
			seenKey["tenantID"] = true
		case strings.HasPrefix(clause, "metadata->>'objectType' = $"):
			seenKey["objectType"] = true
		default:
			t.Fatalf("clause is not a metadata projection: %q", clause)
		}
	}
	for _, a := range args {
		argSet[a.(string)] = true
	}
	if !seenKey["tenantID"] || !seenKey["objectType"] {
		t.Fatalf("missing metadata predicates: %v", where)
	}
	// 值统一文本化：int 2 → "2"（与 ->> 投影语义一致）
	if !argSet["2"] || !argSet["kb"] {
		t.Fatalf("unexpected args: %v", args)
	}
	// 占位符编号从 argOffset 连续递增：真实调用中 $1 是向量，
	// argOffset=1 表示过滤参数从 $2 开始
	joined := strings.Join(where, " AND ")
	if !strings.Contains(joined, "$2") || !strings.Contains(joined, "$3") {
		t.Fatalf("placeholder numbering broken: %v", where)
	}
}

// 非法键 fail closed（键会内嵌进单引号 SQL 文本）
func TestSearchFilterClauses_RejectsInvalidKey(t *testing.T) {
	if _, _, err := searchFilterClauses(map[string]interface{}{"x'); DROP TABLE vectors;--": 1}, 1); err == nil {
		t.Fatal("expected error for invalid identifier key")
	}
	if _, _, err := searchFilterClauses(map[string]interface{}{"tenant id": 1}, 1); err == nil {
		t.Fatal("expected error for key with space")
	}
}

// 空/nil 过滤器不产生谓词
func TestSearchFilterClauses_Empty(t *testing.T) {
	for _, f := range []map[string]interface{}{nil, {}} {
		where, args, err := searchFilterClauses(f, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(where) != 0 || len(args) != 0 {
			t.Fatalf("expected no clauses/args, got %v/%v", where, args)
		}
	}
}
