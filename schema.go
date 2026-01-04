// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 15:15
//
// --------------------------------------------
package difftable

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// getTableSchema 获取单个表的完整结构快照
func (d *Differ) getTableSchema(db *gorm.DB, dbName string, tableName string) (*Table, error) {
	table := &Table{
		Name:        tableName,
		Columns:     make(map[string]Column),
		Indexes:     make(map[string]Index),
		ForeignKeys: make(map[string]ForeignKey),
	}

	// 获取列信息
	if err := d.fetchColumns(db, dbName, tableName, table); err != nil {
		return nil, fmt.Errorf("fetch columns: %w", err)
	}

	// 获取索引信息
	if err := d.fetchIndexes(db, tableName, table); err != nil {
		return nil, fmt.Errorf("fetch indexes: %w", err)
	}

	// 获取主键信息
	if err := d.fetchPrimaryKey(db, tableName, table); err != nil {
		return nil, fmt.Errorf("fetch primary key: %w", err)
	}

	// 获取外键信息
	if err := d.fetchForeignKeys(db, dbName, tableName, table); err != nil {
		return nil, fmt.Errorf("fetch foreign keys: %w", err)
	}

	return table, nil
}

// fetchColumns 通过 GORM ColumnTypes + Raw SQL 获取列信息
func (d *Differ) fetchColumns(db *gorm.DB, dbName string, tableName string, table *Table) error {
	// GORM Migrator 获取基础列信息
	m := db.Migrator()
	if !m.HasTable(tableName) {
		return fmt.Errorf("table %s not found", tableName)
	}

	columnTypes, err := m.ColumnTypes(tableName)
	if err != nil {
		return err
	}

	for _, ct := range columnTypes {
		col := Column{
			Name: ct.Name(),
		}

		// 数据类型
		col.DataType = ct.DatabaseTypeName()

		// 是否可为 NULL
		if nullable, ok := ct.Nullable(); ok {
			col.IsNull = nullable
		}

		// 默认值
		if defaultVal, ok := ct.DefaultValue(); ok && defaultVal != "" {
			col.Default = &defaultVal
		}

		table.Columns[col.Name] = col
	}

	// Raw SQL 补充 Extra 和 Comment（MySQL 通过 information_schema）
	d.fetchColumnExtras(db, dbName, tableName, table)

	return nil
}

// fetchColumnExtras 通过 Raw SQL 补充 Extra 和 Comment
func (d *Differ) fetchColumnExtras(db *gorm.DB, dbName string, tableName string, table *Table) {
	dbNameVal := dbName
	if dbNameVal == "" {
		dbNameVal = db.Migrator().CurrentDatabase()
	}

	// 查询 information_schema 获取 Extra 和 Comment
	type columnExtra struct {
		ColumnName string
		Extra      string
		Comment    string
	}

	var extras []columnExtra
	sql := `SELECT COLUMN_NAME, EXTRA, COLUMN_COMMENT
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`

	if err := db.Raw(sql, dbNameVal, tableName).Scan(&extras).Error; err != nil {
		if d.logger != nil {
			d.logger.Warn("fetchColumnExtras failed, skipping", zap.Error(err))
		}
		return
	}

	for _, extra := range extras {
		if col, ok := table.Columns[extra.ColumnName]; ok {
			col.Extra = extra.Extra
			col.Comment = extra.Comment
			table.Columns[extra.ColumnName] = col
		}
	}
}

// fetchIndexes 通过 GORM Indexes 获取索引信息
func (d *Differ) fetchIndexes(db *gorm.DB, tableName string, table *Table) error {
	m := db.Migrator()
	indexes, err := m.GetIndexes(tableName)
	if err != nil {
		return err
	}

	for _, idx := range indexes {
		unique, _ := idx.Unique()
		isPrimary, _ := idx.PrimaryKey()
		index := Index{
			Name:      idx.Name(),
			Columns:   idx.Columns(),
			IsUnique:  unique,
			IsPrimary: isPrimary,
		}
		table.Indexes[index.Name] = index
	}

	return nil
}

// fetchPrimaryKey 从索引信息中提取主键列
func (d *Differ) fetchPrimaryKey(db *gorm.DB, tableName string, table *Table) error {
	// 从已获取的索引中查找主键索引
	for _, idx := range table.Indexes {
		if idx.IsPrimary {
			table.PrimaryKey = idx.Columns
			return nil
		}
	}

	// 如果索引中没有标记主键，尝试通过 ColumnTypes 推断
	m := db.Migrator()
	columnTypes, err := m.ColumnTypes(tableName)
	if err != nil {
		return err
	}

	for _, ct := range columnTypes {
		if isPK, ok := ct.PrimaryKey(); ok && isPK {
			table.PrimaryKey = append(table.PrimaryKey, ct.Name())
		}
	}

	// 回写主键标记到对应索引
	if len(table.PrimaryKey) > 0 {
		pkSet := make(map[string]bool)
		for _, pk := range table.PrimaryKey {
			pkSet[pk] = true
		}
		for name, idx := range table.Indexes {
			isPrimary := true
			for _, col := range idx.Columns {
				if !pkSet[col] {
					isPrimary = false
					break
				}
			}
			if isPrimary && len(idx.Columns) == len(table.PrimaryKey) {
				idx.IsPrimary = true
				table.Indexes[name] = idx
			}
		}
	}

	return nil
}

// fetchForeignKeys 通过 Raw SQL 获取外键信息
func (d *Differ) fetchForeignKeys(db *gorm.DB, dbName string, tableName string, table *Table) error {
	dbNameVal := dbName
	if dbNameVal == "" {
		dbNameVal = db.Migrator().CurrentDatabase()
	}

	type fkRow struct {
		ConstraintName string
		ColumnName     string
		RefTableName   string
		RefColumnName  string
		OnDelete       string
		OnUpdate       string
	}

	var rows []fkRow
	sql := `SELECT
		kcu.CONSTRAINT_NAME,
		kcu.COLUMN_NAME,
		kcu.REFERENCED_TABLE_NAME,
		kcu.REFERENCED_COLUMN_NAME,
		rc.DELETE_RULE,
		rc.UPDATE_RULE
	FROM information_schema.KEY_COLUMN_USAGE kcu
	JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
		ON kcu.CONSTRAINT_NAME = rc.CONSTRAINT_NAME
		AND kcu.TABLE_SCHEMA = rc.CONSTRAINT_SCHEMA
	WHERE kcu.TABLE_SCHEMA = ?
		AND kcu.TABLE_NAME = ?
		AND kcu.REFERENCED_TABLE_NAME IS NOT NULL`

	if err := db.Raw(sql, dbNameVal, tableName).Scan(&rows).Error; err != nil {
		if d.logger != nil {
			d.logger.Warn("fetchForeignKeys failed, skipping", zap.Error(err))
		}
		return nil
	}

	for _, row := range rows {
		fk := ForeignKey{
			Name:      row.ConstraintName,
			Column:    row.ColumnName,
			RefTable:  row.RefTableName,
			RefColumn: row.RefColumnName,
			OnDelete:  row.OnDelete,
			OnUpdate:  row.OnUpdate,
		}
		table.ForeignKeys[fk.Name] = fk
	}

	return nil
}

// ListSrcTables 扫描源库表列表，受 tbPrefix / tbNames 过滤
func (d *Differ) ListSrcTables() ([]string, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}

	tables, err := d.srcDb.Migrator().GetTables()
	if err != nil {
		return nil, fmt.Errorf("get tables: %w", err)
	}

	return d.filterTables(tables), nil
}

// filterTables 按 tbPrefix 和 tbNames 过滤表名（OR 关系）
func (d *Differ) filterTables(tables []string) []string {
	// 无过滤条件时返回全部
	if len(d.tbPrefix) == 0 && len(d.tbNames) == 0 {
		return tables
	}

	nameSet := make(map[string]bool)
	for _, name := range d.tbNames {
		nameSet[name] = true
	}

	var result []string
	for _, t := range tables {
		// 精确匹配
		if nameSet[t] {
			result = append(result, t)
			continue
		}
		// 前缀匹配
		for _, prefix := range d.tbPrefix {
			if strings.HasPrefix(t, prefix) {
				result = append(result, t)
				break
			}
		}
	}

	return result
}
