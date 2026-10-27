package migrate

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// TestNoTableDeclaresDuplicateIndexes 守住「同一张表不声明两份等价索引」。
//
// 这类重复不会有任何功能报错，只表现为写放大与规划器候选集翻倍，因此只能靠
// 静态对账发现。2026-10-27 生产库实测 32 对完全重复的索引，其中 6 对由 ent
// 自身造成：字段声明了 .Unique()（PostgreSQL 侧生成 <表>_<列>_key 唯一约束），
// 又在 Indexes() 里写了一遍 index.Fields("<列>").Unique()（生成 <实体>_<列>
// 唯一索引）。二者列相同、唯一性相同，是同一约束的两份物化。
//
// **候选集必须同时包含列级唯一约束**：字段上的 .Unique() 不会出现在
// tbl.Indexes 里，只表现为 tbl.Columns[i].Unique == true。只看 Indexes 会漏掉
// 这 6 对而假绿——本测试第一版就是这么写的，反向注入重复索引仍然 PASS。
//
// 判据必须包含唯一性：同列但一唯一一不唯一的索引语义不同（后者允许重复值），
// 只看列名会把「唯一约束索引 + 普通查询索引」误判成重复而错误合并。
//
// 反向证明：向 cmdb_export_tasks 的 Indexes 注入一条 Unique 的 task_id 索引
// （与列级 .Unique() 等价）时本测试报红；删除后通过。
func TestNoTableDeclaresDuplicateIndexes(t *testing.T) {
	if len(Tables) == 0 {
		t.Fatal("Tables 为空：生成物缺失时本守卫会假绿，直接判红")
	}

	type candidate struct {
		name   string
		unique bool
		cols   []string
		// annotation 为空表示无谓词（全量索引）。列级唯一约束恒为全量索引。
		annotation string
	}

	tablesChecked := 0
	candidatesChecked := 0
	columnUniques := 0

	for _, tbl := range Tables {
		if len(tbl.Columns) == 0 {
			continue
		}
		tablesChecked++

		cands := make([]candidate, 0, len(tbl.Columns)+len(tbl.Indexes))
		for _, c := range tbl.Columns {
			if !c.Unique {
				continue
			}
			cands = append(cands, candidate{
				name:   fmt.Sprintf("<column %s UNIQUE>", c.Name),
				unique: true,
				cols:   []string{c.Name},
			})
			columnUniques++
		}
		for _, idx := range tbl.Indexes {
			cols := make([]string, 0, len(idx.Columns))
			for _, c := range idx.Columns {
				cols = append(cols, c.Name)
			}
			// schema.Index.Annotation 是无谓词索引时取 nil 指针，%v 会输出 "<nil>"；
			// 列级唯一约束没有 Annotation 概念，用空串。不归一化的话两把钥匙永远
			// 不同，重复判据一次也命中不了——本测试第一版就栽在这里。
			ann := fmt.Sprintf("%v", idx.Annotation)
			if ann == "<nil>" {
				ann = ""
			}
			cands = append(cands, candidate{
				name:       idx.Name,
				unique:     idx.Unique,
				cols:       cols,
				annotation: ann,
			})
		}

		seen := make(map[string]string, len(cands))
		for _, cand := range cands {
			// fmt 对 map 按 key 排序输出，因此 annotation 字符串在同一进程内稳定。
			key := strings.Join(cand.cols, ",") +
				"|unique=" + strconv.FormatBool(cand.unique) +
				"|annotation=" + cand.annotation

			if prev, dup := seen[key]; dup {
				t.Errorf("%s: %s 与 %s 定义完全相同（列 [%s], unique=%v）——"+
					"二者在查询计划上等价，只增加写放大；保留一条即可",
					tbl.Name, cand.name, prev, strings.Join(cand.cols, ", "), cand.unique)
				continue
			}
			seen[key] = cand.name
			candidatesChecked++
		}
	}

	if tablesChecked < 50 {
		t.Fatalf("只扫到 %d 张表，疑似生成物被截断；守卫拒绝在低覆盖下放行", tablesChecked)
	}
	if candidatesChecked < 200 {
		t.Fatalf("只扫到 %d 条索引/唯一约束，疑似生成物被截断；守卫拒绝在低覆盖下放行", candidatesChecked)
	}
	if columnUniques == 0 {
		t.Fatal("没有扫到任何列级唯一约束：字段级 .Unique() 的解析路径失效，守卫会漏掉这一整类重复")
	}
}
