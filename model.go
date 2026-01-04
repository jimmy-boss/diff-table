// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 15:05
//
// --------------------------------------------
package difftable

// Column 列定义
type Column struct {
	Name     string  `json:"name"`
	DataType string  `json:"data_type"`
	IsNull   bool    `json:"is_null"`
	Default  *string `json:"default"`
	Extra    string  `json:"extra"`
	Comment  string  `json:"comment"`
}

// Index 索引定义
type Index struct {
	Name      string   `json:"name"`
	Columns   []string `json:"columns"`
	IsUnique  bool     `json:"is_unique"`
	IsPrimary bool     `json:"is_primary"`
}

// ForeignKey 外键定义
type ForeignKey struct {
	Name      string `json:"name"`
	Column    string `json:"column"`
	RefTable  string `json:"ref_table"`
	RefColumn string `json:"ref_column"`
	OnDelete  string `json:"on_delete"`
	OnUpdate  string `json:"on_update"`
}

// Table 表结构快照
type Table struct {
	Name        string                `json:"name"`
	Columns     map[string]Column     `json:"columns"`
	Indexes     map[string]Index      `json:"indexes"`
	ForeignKeys map[string]ForeignKey `json:"foreign_keys"`
	PrimaryKey  []string              `json:"primary_key"`
}

// DiffType 差异类型
type DiffType string

const (
	DiffTypeAdded   DiffType = "added"
	DiffTypeRemoved DiffType = "removed"
	DiffTypeChanged DiffType = "changed"
	DiffTypeEqual   DiffType = "equal"
)

// DiffMode 差异模式
type DiffMode string

const (
	DiffOnly     DiffMode = "diff_only"
	FullSnapshot DiffMode = "full_snapshot"
)

// OutputFormat 输出格式
type OutputFormat string

const (
	OutputJSON         OutputFormat = "json"
	OutputConsoleDiff  OutputFormat = "diff"
	OutputConsoleTable OutputFormat = "table"
	OutputConsoleTree  OutputFormat = "tree"
)

// ColumnDiff 单列差异
type ColumnDiff struct {
	Name    string      `json:"name"`
	Type    DiffType    `json:"type"`
	Src     *Column     `json:"src,omitempty"`
	Dst     *Column     `json:"dst,omitempty"`
	Details []FieldDiff `json:"details,omitempty"`
}

// FieldDiff 字段级别的差异明细
type FieldDiff struct {
	Field string      `json:"field"`
	Src   interface{} `json:"src"`
	Dst   interface{} `json:"dst"`
}

// IndexDiff 索引差异
type IndexDiff struct {
	Name string   `json:"name"`
	Type DiffType `json:"type"`
	Src  *Index   `json:"src,omitempty"`
	Dst  *Index   `json:"dst,omitempty"`
}

// FKDiff 外键差异
type FKDiff struct {
	Name string      `json:"name"`
	Type DiffType    `json:"type"`
	Src  *ForeignKey `json:"src,omitempty"`
	Dst  *ForeignKey `json:"dst,omitempty"`
}

// PrimaryKeyDiff 主键差异
type PrimaryKeyDiff struct {
	Type DiffType `json:"type"`
	Src  []string `json:"src,omitempty"`
	Dst  []string `json:"dst,omitempty"`
}

// TableDiff 单表的完整差异
type TableDiff struct {
	TableName   string         `json:"table_name"`
	HasDiff     bool           `json:"has_diff"`
	Columns     []ColumnDiff   `json:"columns"`
	Indexes     []IndexDiff    `json:"indexes"`
	ForeignKeys []FKDiff       `json:"foreign_keys"`
	PrimaryKey  PrimaryKeyDiff `json:"primary_key"`
}

// CompareResult 所有表的对比结果
type CompareResult struct {
	Tables  []TableDiff `json:"tables"`
	Summary Summary     `json:"summary"`
}

// Summary 汇总统计
type Summary struct {
	TotalTables  int `json:"total_tables"`
	DiffTables   int `json:"diff_tables"`
	EqualTables  int `json:"equal_tables"`
	TotalColumns int `json:"total_columns"`
	DiffColumns  int `json:"diff_columns"`
}
