// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 15:30
//
// --------------------------------------------
package difftable

import (
	"bytes"
	"fmt"
	"strings"
)

// DiffFormatter +/- 风格输出
type DiffFormatter struct{}

func (f *DiffFormatter) Format(result *CompareResult) ([]byte, error) {
	var buf bytes.Buffer

	for _, td := range result.Tables {
		fmt.Fprintf(&buf, "=== Table: %s ===\n", td.TableName)

		// Columns
		if len(td.Columns) > 0 {
			buf.WriteString("  Columns:\n")
			for _, cd := range td.Columns {
				switch cd.Type {
				case DiffTypeEqual:
					// 仅 FullSnapshot 模式下显示
				case DiffTypeAdded:
					fmt.Fprintf(&buf, "    + %s: %s\n", cd.Name, formatColumnShort(cd.Dst))
				case DiffTypeRemoved:
					fmt.Fprintf(&buf, "    - %s: %s\n", cd.Name, formatColumnShort(cd.Src))
				case DiffTypeChanged:
					fmt.Fprintf(&buf, "    ~ %s: %s → %s\n", cd.Name, formatColumnShort(cd.Src), formatColumnShort(cd.Dst))
					for _, detail := range cd.Details {
						fmt.Fprintf(&buf, "        %s: %v → %v\n", detail.Field, detail.Src, detail.Dst)
					}
				}
			}
		}

		// Indexes
		hasDiffIndex := false
		for _, id := range td.Indexes {
			if id.Type != DiffTypeEqual {
				if !hasDiffIndex {
					buf.WriteString("  Indexes:\n")
					hasDiffIndex = true
				}
				switch id.Type {
				case DiffTypeAdded:
					fmt.Fprintf(&buf, "    + %s: (%s)%s\n", id.Name, strings.Join(id.Dst.Columns, ", "), uniqueSuffix(id.Dst))
				case DiffTypeRemoved:
					fmt.Fprintf(&buf, "    - %s: (%s)%s\n", id.Name, strings.Join(id.Src.Columns, ", "), uniqueSuffix(id.Src))
				case DiffTypeChanged:
					fmt.Fprintf(&buf, "    ~ %s\n", id.Name)
				}
			}
		}

		// ForeignKeys
		hasDiffFK := false
		for _, fd := range td.ForeignKeys {
			if fd.Type != DiffTypeEqual {
				if !hasDiffFK {
					buf.WriteString("  ForeignKeys:\n")
					hasDiffFK = true
				}
				switch fd.Type {
				case DiffTypeAdded:
					fmt.Fprintf(&buf, "    + %s: %s → %s(%s)\n", fd.Name, fd.Dst.Column, fd.Dst.RefTable, fd.Dst.RefColumn)
				case DiffTypeRemoved:
					fmt.Fprintf(&buf, "    - %s: %s → %s(%s)\n", fd.Name, fd.Src.Column, fd.Src.RefTable, fd.Src.RefColumn)
				case DiffTypeChanged:
					fmt.Fprintf(&buf, "    ~ %s\n", fd.Name)
				}
			}
		}

		// PrimaryKey
		if td.PrimaryKey.Type != DiffTypeEqual {
			fmt.Fprintf(&buf, "  PrimaryKey: [%s] → [%s]\n",
				strings.Join(td.PrimaryKey.Src, ", "),
				strings.Join(td.PrimaryKey.Dst, ", "))
		}

		if !td.HasDiff {
			buf.WriteString("  (no differences)\n")
		}

		buf.WriteString("\n")
	}

	// Summary
	fmt.Fprintf(&buf, "Summary: %d tables, %d differ, %d equal\n",
		result.Summary.TotalTables, result.Summary.DiffTables, result.Summary.EqualTables)

	return buf.Bytes(), nil
}

func formatColumnShort(col *Column) string {
	if col == nil {
		return "-"
	}
	s := col.DataType
	if !col.IsNull {
		s += ", NOT NULL"
	} else {
		s += ", NULL"
	}
	if col.Default != nil {
		s += ", DEFAULT " + *col.Default
	}
	return s
}

func uniqueSuffix(idx *Index) string {
	if idx == nil {
		return ""
	}
	if idx.IsUnique {
		return " UNIQUE"
	}
	if idx.IsPrimary {
		return " PRIMARY"
	}
	return ""
}
