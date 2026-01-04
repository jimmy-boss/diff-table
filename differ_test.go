// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 16:00
//
// --------------------------------------------
package difftable

import (
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/gorm"
)

// ---------- helpers ----------

func strPtr(s string) *string {
	return &s
}

func buildTable(name string, cols []Column, indexes []Index, fks []ForeignKey, pk []string) *Table {
	t := &Table{
		Name:        name,
		Columns:     make(map[string]Column),
		Indexes:     make(map[string]Index),
		ForeignKeys: make(map[string]ForeignKey),
		PrimaryKey:  pk,
	}
	for _, c := range cols {
		t.Columns[c.Name] = c
	}
	for _, idx := range indexes {
		t.Indexes[idx.Name] = idx
	}
	for _, fk := range fks {
		t.ForeignKeys[fk.Name] = fk
	}
	return t
}

// newFullDiffer 创建一个 FullSnapshot 模式的 Differ（对比时保留 equal 列）
func newFullDiffer() *Differ {
	return NewDiffer(WithDiffMode(FullSnapshot))
}

// ---------- 1. CompareColumns_Equal ----------

func TestCompareColumns_Equal(t *testing.T) {
	cols := []Column{
		{Name: "id", DataType: "bigint", IsNull: false, Extra: "auto_increment", Comment: "主键"},
		{Name: "name", DataType: "varchar(255)", IsNull: true, Comment: "名称"},
	}
	src := buildTable("users", cols, nil, nil, []string{"id"})
	dst := buildTable("users", cols, nil, nil, []string{"id"})

	d := newFullDiffer()
	diffs := d.compareColumns(src, dst)

	if len(diffs) != 2 {
		t.Fatalf("expected 2 column diffs, got %d", len(diffs))
	}
	for _, cd := range diffs {
		if cd.Type != DiffTypeEqual {
			t.Errorf("column %s: expected equal, got %s", cd.Name, cd.Type)
		}
	}
}

// ---------- 2. CompareColumns_Added ----------

func TestCompareColumns_Added(t *testing.T) {
	srcCols := []Column{
		{Name: "id", DataType: "bigint", IsNull: false},
	}
	dstCols := []Column{
		{Name: "id", DataType: "bigint", IsNull: false},
		{Name: "email", DataType: "varchar(100)", IsNull: true},
	}
	src := buildTable("users", srcCols, nil, nil, []string{"id"})
	dst := buildTable("users", dstCols, nil, nil, []string{"id"})

	d := newFullDiffer()
	diffs := d.compareColumns(src, dst)

	found := false
	for _, cd := range diffs {
		if cd.Name == "email" && cd.Type == DiffTypeAdded {
			found = true
			if cd.Dst == nil {
				t.Error("added column should have Dst set")
			}
			if cd.Src != nil {
				t.Error("added column should not have Src set")
			}
		}
	}
	if !found {
		t.Error("expected to find added column 'email'")
	}

	td := d.compareTables(src, dst)
	if !td.HasDiff {
		t.Error("expected HasDiff=true when column added")
	}
}

// ---------- 3. CompareColumns_Removed ----------

func TestCompareColumns_Removed(t *testing.T) {
	srcCols := []Column{
		{Name: "id", DataType: "bigint", IsNull: false},
		{Name: "email", DataType: "varchar(100)", IsNull: true},
	}
	dstCols := []Column{
		{Name: "id", DataType: "bigint", IsNull: false},
	}
	src := buildTable("users", srcCols, nil, nil, []string{"id"})
	dst := buildTable("users", dstCols, nil, nil, []string{"id"})

	d := newFullDiffer()
	diffs := d.compareColumns(src, dst)

	found := false
	for _, cd := range diffs {
		if cd.Name == "email" && cd.Type == DiffTypeRemoved {
			found = true
			if cd.Src == nil {
				t.Error("removed column should have Src set")
			}
			if cd.Dst != nil {
				t.Error("removed column should not have Dst set")
			}
		}
	}
	if !found {
		t.Error("expected to find removed column 'email'")
	}

	td := d.compareTables(src, dst)
	if !td.HasDiff {
		t.Error("expected HasDiff=true when column removed")
	}
}

// ---------- 4. CompareColumns_TypeChanged ----------

func TestCompareColumns_TypeChanged(t *testing.T) {
	srcCols := []Column{
		{Name: "id", DataType: "bigint", IsNull: false},
		{Name: "age", DataType: "int", IsNull: true},
	}
	dstCols := []Column{
		{Name: "id", DataType: "bigint", IsNull: false},
		{Name: "age", DataType: "bigint", IsNull: true},
	}
	src := buildTable("users", srcCols, nil, nil, []string{"id"})
	dst := buildTable("users", dstCols, nil, nil, []string{"id"})

	d := newFullDiffer()
	diffs := d.compareColumns(src, dst)

	found := false
	for _, cd := range diffs {
		if cd.Name == "age" && cd.Type == DiffTypeChanged {
			found = true
			if len(cd.Details) == 0 {
				t.Error("changed column should have details")
			}
			hasDataType := false
			for _, detail := range cd.Details {
				if detail.Field == "DataType" {
					hasDataType = true
					if detail.Src != "int" {
						t.Errorf("expected src DataType 'int', got %v", detail.Src)
					}
					if detail.Dst != "bigint" {
						t.Errorf("expected dst DataType 'bigint', got %v", detail.Dst)
					}
				}
			}
			if !hasDataType {
				t.Error("expected DataType field in details")
			}
		}
	}
	if !found {
		t.Error("expected to find changed column 'age'")
	}
}

// ---------- 5. CompareColumns_DefaultChanged ----------

func TestCompareColumns_DefaultChanged(t *testing.T) {
	srcCols := []Column{
		{Name: "id", DataType: "bigint", IsNull: false},
		{Name: "status", DataType: "int", IsNull: false, Default: strPtr("0")},
	}
	dstCols := []Column{
		{Name: "id", DataType: "bigint", IsNull: false},
		{Name: "status", DataType: "int", IsNull: false, Default: strPtr("1")},
	}
	src := buildTable("users", srcCols, nil, nil, []string{"id"})
	dst := buildTable("users", dstCols, nil, nil, []string{"id"})

	d := newFullDiffer()
	diffs := d.compareColumns(src, dst)

	found := false
	for _, cd := range diffs {
		if cd.Name == "status" && cd.Type == DiffTypeChanged {
			found = true
			hasDefault := false
			for _, detail := range cd.Details {
				if detail.Field == "Default" {
					hasDefault = true
					srcVal, _ := detail.Src.(*string)
					dstVal, _ := detail.Dst.(*string)
					if srcVal == nil || *srcVal != "0" {
						t.Errorf("expected src default '0', got %v", detail.Src)
					}
					if dstVal == nil || *dstVal != "1" {
						t.Errorf("expected dst default '1', got %v", detail.Dst)
					}
				}
			}
			if !hasDefault {
				t.Error("expected Default field in details")
			}
		}
	}
	if !found {
		t.Error("expected to find changed column 'status' with default diff")
	}
}

// ---------- 6. CompareIndexes_Added ----------

func TestCompareIndexes_Added(t *testing.T) {
	srcIdx := []Index{
		{Name: "idx_name", Columns: []string{"name"}, IsUnique: false},
	}
	dstIdx := []Index{
		{Name: "idx_name", Columns: []string{"name"}, IsUnique: false},
		{Name: "idx_email", Columns: []string{"email"}, IsUnique: true},
	}
	src := buildTable("users", nil, srcIdx, nil, nil)
	dst := buildTable("users", nil, dstIdx, nil, nil)

	diffs := compareIndexes(src, dst)

	found := false
	for _, id := range diffs {
		if id.Name == "idx_email" && id.Type == DiffTypeAdded {
			found = true
			if id.Dst == nil {
				t.Error("added index should have Dst set")
			}
			if id.Src != nil {
				t.Error("added index should not have Src set")
			}
		}
	}
	if !found {
		t.Error("expected to find added index 'idx_email'")
	}
}

// ---------- 7. CompareForeignKeys_Removed ----------

func TestCompareForeignKeys_Removed(t *testing.T) {
	srcFKs := []ForeignKey{
		{Name: "fk_user_dept", Column: "dept_id", RefTable: "departments", RefColumn: "id", OnDelete: "CASCADE", OnUpdate: "NO ACTION"},
	}
	src := buildTable("users", nil, nil, srcFKs, nil)
	dst := buildTable("users", nil, nil, nil, nil)

	diffs := compareFKs(src, dst)

	found := false
	for _, fd := range diffs {
		if fd.Name == "fk_user_dept" && fd.Type == DiffTypeRemoved {
			found = true
			if fd.Src == nil {
				t.Error("removed FK should have Src set")
			}
			if fd.Dst != nil {
				t.Error("removed FK should not have Dst set")
			}
		}
	}
	if !found {
		t.Error("expected to find removed FK 'fk_user_dept'")
	}
}

// ---------- 8. ComparePrimaryKey_Changed ----------

func TestComparePrimaryKey_Changed(t *testing.T) {
	src := buildTable("users", nil, nil, nil, []string{"id"})
	dst := buildTable("users", nil, nil, nil, []string{"id", "tenant_id"})

	pkDiff := comparePrimaryKey(src, dst)

	if pkDiff.Type != DiffTypeChanged {
		t.Errorf("expected PrimaryKeyDiff type changed, got %s", pkDiff.Type)
	}
	if len(pkDiff.Src) != 1 || pkDiff.Src[0] != "id" {
		t.Errorf("expected src pk [id], got %v", pkDiff.Src)
	}
	if len(pkDiff.Dst) != 2 || pkDiff.Dst[0] != "id" || pkDiff.Dst[1] != "tenant_id" {
		t.Errorf("expected dst pk [id, tenant_id], got %v", pkDiff.Dst)
	}
}

// ---------- 9. CompareEmptyTables ----------

func TestCompareEmptyTables(t *testing.T) {
	src := buildTable("empty_table", nil, nil, nil, nil)
	dst := buildTable("empty_table", nil, nil, nil, nil)

	d := newFullDiffer()
	td := d.compareTables(src, dst)

	if td.HasDiff {
		t.Error("expected HasDiff=false for two empty tables")
	}
	if len(td.Columns) != 0 {
		t.Errorf("expected 0 columns, got %d", len(td.Columns))
	}
	if len(td.Indexes) != 0 {
		t.Errorf("expected 0 indexes, got %d", len(td.Indexes))
	}
	if len(td.ForeignKeys) != 0 {
		t.Errorf("expected 0 foreign keys, got %d", len(td.ForeignKeys))
	}
	if td.PrimaryKey.Type != DiffTypeEqual {
		t.Errorf("expected primary key equal, got %s", td.PrimaryKey.Type)
	}
}

// ---------- 10. FilterTables ----------

func TestFilterTables(t *testing.T) {
	tables := []string{"sys_users", "sys_roles", "orders", "products", "user_logs"}

	// prefix filter
	d1 := NewDiffer(WithTbPrefix("sys_"))
	result1 := d1.filterTables(tables)
	if len(result1) != 2 {
		t.Errorf("expected 2 tables with prefix 'sys_', got %d: %v", len(result1), result1)
	}

	// exact name filter
	d2 := NewDiffer(WithTbName("orders", "products"))
	result2 := d2.filterTables(tables)
	if len(result2) != 2 {
		t.Errorf("expected 2 tables with exact names, got %d: %v", len(result2), result2)
	}

	// OR logic: prefix + exact name
	d3 := NewDiffer(WithTbPrefix("sys_"), WithTbName("orders"))
	result3 := d3.filterTables(tables)
	if len(result3) != 3 {
		t.Errorf("expected 3 tables with prefix+name OR, got %d: %v", len(result3), result3)
	}

	// no match
	d4 := NewDiffer(WithTbPrefix("nonexist_"))
	result4 := d4.filterTables(tables)
	if len(result4) != 0 {
		t.Errorf("expected 0 tables, got %d: %v", len(result4), result4)
	}
}

// ---------- 11. FilterTables_NoFilter ----------

func TestFilterTables_NoFilter(t *testing.T) {
	tables := []string{"users", "orders", "products"}
	d := NewDiffer()
	result := d.filterTables(tables)

	if len(result) != 3 {
		t.Errorf("expected 3 tables (no filter), got %d", len(result))
	}
	for i, name := range result {
		if name != tables[i] {
			t.Errorf("expected table[%d]=%s, got %s", i, tables[i], name)
		}
	}
}

// ---------- 12. JSONFormatter ----------

func TestJSONFormatter(t *testing.T) {
	result := &CompareResult{
		Tables: []TableDiff{
			{
				TableName: "users",
				HasDiff:   true,
				Columns: []ColumnDiff{
					{Name: "age", Type: DiffTypeChanged, Src: &Column{Name: "age", DataType: "int"}, Dst: &Column{Name: "age", DataType: "bigint"}},
				},
			},
		},
		Summary: Summary{TotalTables: 1, DiffTables: 1, EqualTables: 0},
	}

	f := &JSONFormatter{}
	data, err := f.Format(result)
	if err != nil {
		t.Fatalf("JSONFormatter.Format error: %v", err)
	}

	// roundtrip: unmarshal and verify
	var roundtrip CompareResult
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if len(roundtrip.Tables) != 1 {
		t.Fatalf("expected 1 table in roundtrip, got %d", len(roundtrip.Tables))
	}
	if roundtrip.Tables[0].TableName != "users" {
		t.Errorf("expected table name 'users', got '%s'", roundtrip.Tables[0].TableName)
	}
	if roundtrip.Tables[0].Columns[0].Type != DiffTypeChanged {
		t.Errorf("expected column type 'changed', got '%s'", roundtrip.Tables[0].Columns[0].Type)
	}
}

// ---------- 13. DiffFormatter ----------

func TestDiffFormatter(t *testing.T) {
	result := &CompareResult{
		Tables: []TableDiff{
			{
				TableName: "users",
				HasDiff:   true,
				Columns: []ColumnDiff{
					{Name: "email", Type: DiffTypeAdded, Dst: &Column{Name: "email", DataType: "varchar(100)", IsNull: true}},
					{Name: "age", Type: DiffTypeChanged, Src: &Column{Name: "age", DataType: "int", IsNull: true}, Dst: &Column{Name: "age", DataType: "bigint", IsNull: true}},
				},
			},
		},
		Summary: Summary{TotalTables: 1, DiffTables: 1},
	}

	f := &DiffFormatter{}
	data, err := f.Format(result)
	if err != nil {
		t.Fatalf("DiffFormatter.Format error: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "+ ") {
		t.Error("diff output should contain '+' marker for added column")
	}
	if !strings.Contains(output, "~ ") {
		t.Error("diff output should contain '~' marker for changed column")
	}
	if !strings.Contains(output, "Table: users") {
		t.Error("diff output should contain table name header")
	}
}

// ---------- 14. TableFormatter ----------

func TestTableFormatter(t *testing.T) {
	result := &CompareResult{
		Tables: []TableDiff{
			{
				TableName: "orders",
				HasDiff:   false,
				Columns: []ColumnDiff{
					{Name: "id", Type: DiffTypeEqual, Src: &Column{Name: "id", DataType: "bigint"}, Dst: &Column{Name: "id", DataType: "bigint"}},
				},
			},
		},
		Summary: Summary{TotalTables: 1, EqualTables: 1},
	}

	f := &TableFormatter{}
	data, err := f.Format(result)
	if err != nil {
		t.Fatalf("TableFormatter.Format error: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "Name") {
		t.Error("table output should contain 'Name' header")
	}
	if !strings.Contains(output, "Src") {
		t.Error("table output should contain 'Src' header")
	}
	if !strings.Contains(output, "Dst") {
		t.Error("table output should contain 'Dst' header")
	}
	if !strings.Contains(output, "(no differences)") {
		t.Error("table output should contain '(no differences)' for equal table")
	}
}

// ---------- 15. TreeFormatter ----------

func TestTreeFormatter(t *testing.T) {
	result := &CompareResult{
		Tables: []TableDiff{
			{
				TableName: "products",
				HasDiff:   true,
				Columns: []ColumnDiff{
					{Name: "id", Type: DiffTypeEqual, Src: &Column{Name: "id", DataType: "bigint"}, Dst: &Column{Name: "id", DataType: "bigint"}},
					{Name: "price", Type: DiffTypeChanged, Src: &Column{Name: "price", DataType: "decimal(10,2)"}, Dst: &Column{Name: "price", DataType: "decimal(12,4)"}},
				},
				PrimaryKey: PrimaryKeyDiff{Type: DiffTypeEqual, Src: []string{"id"}, Dst: []string{"id"}},
			},
		},
		Summary: Summary{TotalTables: 1, DiffTables: 1},
	}

	f := &TreeFormatter{}
	data, err := f.Format(result)
	if err != nil {
		t.Fatalf("TreeFormatter.Format error: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "├──") {
		t.Error("tree output should contain tree branch character")
	}
	if !strings.Contains(output, "└──") {
		t.Error("tree output should contain tree leaf character")
	}
	if !strings.Contains(output, "│") {
		t.Error("tree output should contain tree vertical line")
	}
	if !strings.Contains(output, "[DIFF]") {
		t.Error("tree output should contain '[DIFF]' status")
	}
	if !strings.Contains(output, "Table: products") {
		t.Error("tree output should contain table name")
	}
}

// ---------- 16. Validate_MissingSrcDb ----------

func TestValidate_MissingSrcDb(t *testing.T) {
	d := NewDiffer()
	err := d.validate()
	if err == nil {
		t.Fatal("expected error when srcDb is nil")
	}
	if !strings.Contains(err.Error(), "srcDb is required") {
		t.Errorf("expected 'srcDb is required' in error, got: %s", err.Error())
	}
}

// ---------- 17. Validate_MissingDstDb ----------

func TestValidate_MissingDstDb(t *testing.T) {
	// srcDb 非 nil，dstDb 为 nil → 应报 dstDb 错误
	d := NewDiffer(WithSrcDb(&gorm.DB{}))
	err := d.validate()
	if err == nil {
		t.Fatal("expected error when dstDb is nil")
	}
	if !strings.Contains(err.Error(), "dstDb is required") {
		t.Errorf("expected 'dstDb is required' in error, got: %s", err.Error())
	}
}
