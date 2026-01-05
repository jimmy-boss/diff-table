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

// fetchColumns 通过 information_schema.COLUMNS 获取列信息（绕过 GORM Migrator 避免 Doris 不支持 LIMIT ?）
func (d *Differ) fetchColumns(db *gorm.DB, dbName string, tableName string, table *Table) error {
	dbNameVal := dbName
	if dbNameVal == "" {
		dbNameVal = db.Migrator().CurrentDatabase()
	}

	// 先校验表是否存在
	if !db.Migrator().HasTable(tableName) {
		return fmt.Errorf("table %s not found", tableName)
	}

	type columnRow struct {
		ColumnName    string
		DataType      string
		IsNullable    string
		ColumnDefault *string
		Extra         string
		ColumnComment string
	}

	var rows []columnRow
	sql := `SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT, EXTRA, COLUMN_COMMENT
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`

	if err := db.Raw(sql, dbNameVal, tableName).Scan(&rows).Error; err != nil {
		return fmt.Errorf("query information_schema.COLUMNS: %w", err)
	}

	for _, row := range rows {
		col := Column{
			Name:     row.ColumnName,
			DataType: row.DataType,
			IsNull:   row.IsNullable == "YES",
			Extra:    row.Extra,
			Comment:  row.ColumnComment,
		}
		if row.ColumnDefault != nil && *row.ColumnDefault != "" {
			col.Default = row.ColumnDefault
		}
		table.Columns[col.Name] = col
	}

	return nil
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

	// 如果索引中没有标记主键，尝试通过 information_schema 推断
	dbName := db.Migrator().CurrentDatabase()
	type pkRow struct {
		ColumnName string
	}
	var rows []pkRow
	sql := `SELECT COLUMN_NAME
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_KEY = 'PRI'
		ORDER BY ORDINAL_POSITION`

	if err := db.Raw(sql, dbName, tableName).Scan(&rows).Error; err != nil {
		return fmt.Errorf("query primary key from information_schema: %w", err)
	}

	for _, row := range rows {
		table.PrimaryKey = append(table.PrimaryKey, row.ColumnName)
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
