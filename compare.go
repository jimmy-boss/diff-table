// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 15:20
//
// --------------------------------------------
package difftable

import "fmt"

// CompareAll 对比所有匹配的表
func (d *Differ) CompareAll() (*CompareResult, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}

	tables, err := d.ListSrcTables()
	if err != nil {
		return nil, fmt.Errorf("list src tables: %w", err)
	}

	result := &CompareResult{}
	for _, tableName := range tables {
		diff, err := d.CompareOne(tableName)
		if err != nil {
			return nil, fmt.Errorf("compare table %s: %w", tableName, err)
		}
		result.Tables = append(result.Tables, *diff)
	}

	// 汇总统计
	for _, td := range result.Tables {
		result.Summary.TotalTables++
		if td.HasDiff {
			result.Summary.DiffTables++
		} else {
			result.Summary.EqualTables++
		}
		result.Summary.TotalColumns += len(td.Columns)
		for _, cd := range td.Columns {
			if cd.Type != DiffTypeEqual {
				result.Summary.DiffColumns++
			}
		}
	}

	return result, nil
}

// CompareOne 对比单个表
func (d *Differ) CompareOne(tableName string) (*TableDiff, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}

	srcDbName := d.srcDbName
	dstDbName := d.dstDbName

	srcTable, err := d.getTableSchema(d.srcDb, srcDbName, tableName)
	if err != nil {
		return nil, fmt.Errorf("get src schema for %s: %w", tableName, err)
	}

	dstTable, err := d.getTableSchema(d.dstDb, dstDbName, tableName)
	if err != nil {
		// dst 表不存在，视为 src 独有
		return &TableDiff{
			TableName:   tableName,
			HasDiff:     true,
			Columns:     diffColumnsAsRemoved(srcTable),
			Indexes:     diffIndexesAsRemoved(srcTable),
			ForeignKeys: diffFKsAsRemoved(srcTable),
			PrimaryKey: PrimaryKeyDiff{
				Type: DiffTypeRemoved,
				Src:  srcTable.PrimaryKey,
			},
		}, nil
	}

	return d.compareTables(srcTable, dstTable), nil
}

// compareTables 对比两个表结构
func (d *Differ) compareTables(src, dst *Table) *TableDiff {
	td := &TableDiff{
		TableName:   src.Name,
		Columns:     d.compareColumns(src, dst),
		Indexes:     compareIndexes(src, dst),
		ForeignKeys: compareFKs(src, dst),
		PrimaryKey:  comparePrimaryKey(src, dst),
	}

	// 判断是否有差异
	td.HasDiff = hasAnyDiff(td)

	return td
}

// compareColumns 对比列
func (d *Differ) compareColumns(src, dst *Table) []ColumnDiff {
	var diffs []ColumnDiff
	seen := make(map[string]bool)

	for name, srcCol := range src.Columns {
		seen[name] = true
		dstCol, exists := dst.Columns[name]
		if !exists {
			diffs = append(diffs, ColumnDiff{
				Name: name,
				Type: DiffTypeRemoved,
				Src:  &srcCol,
			})
			continue
		}

		cd := ColumnDiff{
			Name: name,
			Src:  &srcCol,
			Dst:  &dstCol,
		}

		if columnEqual(srcCol, dstCol) {
			cd.Type = DiffTypeEqual
		} else {
			cd.Type = DiffTypeChanged
			cd.Details = columnFieldDiffs(srcCol, dstCol)
		}

		diffs = append(diffs, cd)
	}

	for name, dstCol := range dst.Columns {
		if !seen[name] {
			diffs = append(diffs, ColumnDiff{
				Name: name,
				Type: DiffTypeAdded,
				Dst:  &dstCol,
			})
		}
	}

	// DiffOnly 模式下过滤掉 equal 的列
	if d.diffMode == DiffOnly {
		filtered := make([]ColumnDiff, 0, len(diffs))
		for _, cd := range diffs {
			if cd.Type != DiffTypeEqual {
				filtered = append(filtered, cd)
			}
		}
		return filtered
	}

	return diffs
}

// columnEqual 判断两列是否完全相同
func columnEqual(a, b Column) bool {
	if a.DataType != b.DataType {
		return false
	}
	if a.IsNull != b.IsNull {
		return false
	}
	if !ptrEqual(a.Default, b.Default) {
		return false
	}
	if a.Extra != b.Extra {
		return false
	}
	if a.Comment != b.Comment {
		return false
	}
	return true
}

func ptrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// columnFieldDiffs 生成列的字段级差异列表
func columnFieldDiffs(src, dst Column) []FieldDiff {
	var diffs []FieldDiff

	if src.DataType != dst.DataType {
		diffs = append(diffs, FieldDiff{Field: "DataType", Src: src.DataType, Dst: dst.DataType})
	}
	if src.IsNull != dst.IsNull {
		diffs = append(diffs, FieldDiff{Field: "IsNull", Src: src.IsNull, Dst: dst.IsNull})
	}
	if !ptrEqual(src.Default, dst.Default) {
		diffs = append(diffs, FieldDiff{Field: "Default", Src: src.Default, Dst: dst.Default})
	}
	if src.Extra != dst.Extra {
		diffs = append(diffs, FieldDiff{Field: "Extra", Src: src.Extra, Dst: dst.Extra})
	}
	if src.Comment != dst.Comment {
		diffs = append(diffs, FieldDiff{Field: "Comment", Src: src.Comment, Dst: dst.Comment})
	}

	return diffs
}

// compareIndexes 对比索引
func compareIndexes(src, dst *Table) []IndexDiff {
	var diffs []IndexDiff
	seen := make(map[string]bool)

	for name, srcIdx := range src.Indexes {
		seen[name] = true
		dstIdx, exists := dst.Indexes[name]
		if !exists {
			diffs = append(diffs, IndexDiff{Name: name, Type: DiffTypeRemoved, Src: &srcIdx})
			continue
		}

		id := IndexDiff{Name: name, Src: &srcIdx, Dst: &dstIdx}
		if indexEqual(srcIdx, dstIdx) {
			id.Type = DiffTypeEqual
		} else {
			id.Type = DiffTypeChanged
		}
		diffs = append(diffs, id)
	}

	for name, dstIdx := range dst.Indexes {
		if !seen[name] {
			diffs = append(diffs, IndexDiff{Name: name, Type: DiffTypeAdded, Dst: &dstIdx})
		}
	}

	return diffs
}

func indexEqual(a, b Index) bool {
	if a.IsUnique != b.IsUnique || a.IsPrimary != b.IsPrimary {
		return false
	}
	if len(a.Columns) != len(b.Columns) {
		return false
	}
	for i, col := range a.Columns {
		if col != b.Columns[i] {
			return false
		}
	}
	return true
}

// compareFKs 对比外键
func compareFKs(src, dst *Table) []FKDiff {
	var diffs []FKDiff
	seen := make(map[string]bool)

	for name, srcFk := range src.ForeignKeys {
		seen[name] = true
		dstFk, exists := dst.ForeignKeys[name]
		if !exists {
			diffs = append(diffs, FKDiff{Name: name, Type: DiffTypeRemoved, Src: &srcFk})
			continue
		}

		fd := FKDiff{Name: name, Src: &srcFk, Dst: &dstFk}
		if fkEqual(srcFk, dstFk) {
			fd.Type = DiffTypeEqual
		} else {
			fd.Type = DiffTypeChanged
		}
		diffs = append(diffs, fd)
	}

	for name, dstFk := range dst.ForeignKeys {
		if !seen[name] {
			diffs = append(diffs, FKDiff{Name: name, Type: DiffTypeAdded, Dst: &dstFk})
		}
	}

	return diffs
}

func fkEqual(a, b ForeignKey) bool {
	return a.Column == b.Column &&
		a.RefTable == b.RefTable &&
		a.RefColumn == b.RefColumn &&
		a.OnDelete == b.OnDelete &&
		a.OnUpdate == b.OnUpdate
}

// comparePrimaryKey 对比主键
func comparePrimaryKey(src, dst *Table) PrimaryKeyDiff {
	if sliceEqual(src.PrimaryKey, dst.PrimaryKey) {
		return PrimaryKeyDiff{Type: DiffTypeEqual, Src: src.PrimaryKey, Dst: dst.PrimaryKey}
	}
	return PrimaryKeyDiff{Type: DiffTypeChanged, Src: src.PrimaryKey, Dst: dst.PrimaryKey}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// diffColumnsAsRemoved 将 src 所有列标记为 removed
func diffColumnsAsRemoved(src *Table) []ColumnDiff {
	var diffs []ColumnDiff
	for name, col := range src.Columns {
		diffs = append(diffs, ColumnDiff{Name: name, Type: DiffTypeRemoved, Src: &col})
	}
	return diffs
}

// diffIndexesAsRemoved 将 src 所有索引标记为 removed
func diffIndexesAsRemoved(src *Table) []IndexDiff {
	var diffs []IndexDiff
	for name, idx := range src.Indexes {
		diffs = append(diffs, IndexDiff{Name: name, Type: DiffTypeRemoved, Src: &idx})
	}
	return diffs
}

// diffFKsAsRemoved 将 src 所有外键标记为 removed
func diffFKsAsRemoved(src *Table) []FKDiff {
	var diffs []FKDiff
	for name, fk := range src.ForeignKeys {
		diffs = append(diffs, FKDiff{Name: name, Type: DiffTypeRemoved, Src: &fk})
	}
	return diffs
}

// hasAnyDiff 判断 TableDiff 是否包含任何差异
func hasAnyDiff(td *TableDiff) bool {
	if td.PrimaryKey.Type != DiffTypeEqual {
		return true
	}
	for _, c := range td.Columns {
		if c.Type != DiffTypeEqual {
			return true
		}
	}
	for _, i := range td.Indexes {
		if i.Type != DiffTypeEqual {
			return true
		}
	}
	for _, f := range td.ForeignKeys {
		if f.Type != DiffTypeEqual {
			return true
		}
	}
	return false
}
