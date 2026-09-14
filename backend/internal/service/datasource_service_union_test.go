package service

import (
	"testing"

	"nextmeta-backend/internal/model"
	"vitess.io/vitess/go/vt/sqlparser"
)

func TestAnalyzeUnionMergesBothBranches(t *testing.T) {
	s := &dataSourceService{}
	rules := []model.DataSourceMaskingRule{
		{Pattern: "*user_account_id*", RuleType: "mask_all"},
	}

	cases := []struct {
		name    string
		sql     string
		wantIdx int
	}{
		{
			name:    "敏感字段只在右分支(常量列名覆盖)",
			sql:     "SELECT 'x','x' FROM orders WHERE 1=0 UNION SELECT order_no, user_account_id FROM orders",
			wantIdx: 1,
		},
		{
			name:    "敏感字段只在左分支",
			sql:     "SELECT user_account_id, 'x' FROM orders UNION SELECT 'a','b' FROM orders",
			wantIdx: 0,
		},
		{
			name:    "左右分支都敏感",
			sql:     "SELECT user_account_id FROM orders UNION SELECT user_account_id FROM orders",
			wantIdx: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser := sqlparser.NewTestParser()
			stmt, err := parser.Parse(tc.sql)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			union, ok := stmt.(*sqlparser.Union)
			if !ok {
				t.Fatalf("expected *sqlparser.Union, got %T", stmt)
			}
			m := s.analyzeUnion(union, rules)
			if _, ok := m[tc.wantIdx]; !ok {
				t.Fatalf("expected masking rule at idx %d, got map %v", tc.wantIdx, m)
			}
		})
	}
}

// TestAnalyzeLineageJoinAndSetOps 覆盖 JOIN LATERAL、JOIN USING、INTERSECT、EXCEPT 四类绕过变种。
// 这些场景的敏感字段被藏进 JOIN 派生表或集合运算分支，血缘分析必须递归或降级为 UNION 才能命中。
func TestAnalyzeLineageJoinAndSetOps(t *testing.T) {
	s := &dataSourceService{}
	rules := []model.DataSourceMaskingRule{
		{Pattern: "user_account_id", RuleType: "mask_all"},
	}

	cases := []struct {
		name    string
		sql     string
		wantIdx int
	}{
		{
			name:    "JOIN LATERAL 派生表",
			sql:     "SELECT x AS a, o.id FROM orders o JOIN LATERAL (SELECT o.user_account_id AS x) t ON 1=1",
			wantIdx: 0,
		},
		{
			name:    "JOIN USING 左侧子查询",
			sql:     "SELECT a FROM (SELECT id, user_account_id AS a FROM orders) x JOIN orders y USING(id)",
			wantIdx: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser := sqlparser.NewTestParser()
			stmt, err := parseQueryForMasking(parser, tc.sql)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			sel, ok := stmt.(*sqlparser.Select)
			if !ok {
				t.Fatalf("expected *sqlparser.Select, got %T", stmt)
			}
			m := s.analyzeLineage(sel, rules)
			if _, ok := m[tc.wantIdx]; !ok {
				t.Fatalf("expected masking rule at idx %d, got map %v", tc.wantIdx, m)
			}
		})
	}
}

// TestParseQueryForMaskingSetOps 验证 INTERSECT / EXCEPT 解析失败时会降级为 UNION 解析。
func TestParseQueryForMaskingSetOps(t *testing.T) {
	s := &dataSourceService{}
	rules := []model.DataSourceMaskingRule{
		{Pattern: "user_account_id", RuleType: "mask_all"},
	}

	cases := []struct {
		name    string
		sql     string
		wantIdx int
	}{
		{
			name:    "INTERSECT 左分支敏感",
			sql:     "SELECT user_account_id AS a FROM orders INTERSECT SELECT user_account_id FROM orders",
			wantIdx: 0,
		},
		{
			name:    "EXCEPT 左分支敏感",
			sql:     "SELECT user_account_id AS a FROM orders EXCEPT SELECT order_no FROM orders",
			wantIdx: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser := sqlparser.NewTestParser()
			stmt, err := parseQueryForMasking(parser, tc.sql)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			union, ok := stmt.(*sqlparser.Union)
			if !ok {
				t.Fatalf("expected *sqlparser.Union after fallback, got %T", stmt)
			}
			m := s.analyzeUnion(union, rules)
			if _, ok := m[tc.wantIdx]; !ok {
				t.Fatalf("expected masking rule at idx %d, got map %v", tc.wantIdx, m)
			}
		})
	}
}
