// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 15:35
//
// --------------------------------------------
package difftable

import (
	"bytes"
	"fmt"
	"strings"
)

// TableFormatter 表格对比输出
type TableFormatter struct{}

func (f *TableFormatter) Format(result *CompareResult) ([]byte, error) {
	var buf bytes.Buffer

	for _, td := range result.Tables {
		fmt.Fprintf(&buf, "=== Table: %s ===\n", td.TableName)

		// Columns
		if len(td.Columns) > 0 {
			buf.WriteString("  Columns:\n")
			nameW, srcW, dstW := calcColumnWidths(td.Columns)
			fmt.Fprintf(&buf, "    %-*s | %-*s | %-*s\n", nameW, "Name", srcW, "Src", dstW, "Dst")
			fmt.Fprintf(&buf, "    %s-+-%s-+-%s\n", strings.Repeat("-", nameW), strings.Repeat("-", srcW), strings.Repeat("-", dstW))
			for _, cd := range td.Columns {
				srcStr := "-"
				dstStr := "-"
				if cd.Src != nil {
					srcStr = formatColumnShort(cd.Src)
				}
				if cd.Dst != nil {
					dstStr = formatColumnShort(cd.Dst)
				}
				fmt.Fprintf(&buf, "    %-*s | %-*s | %-*s\n", nameW, cd.Name, srcW, srcStr, dstW, dstStr)
			}
		}

		// Indexes
		if len(td.Indexes) > 0 {
			buf.WriteString("  Indexes:\n")
			nameW, srcW, dstW := calcIndexWidths(td.Indexes)
			fmt.Fprintf(&buf, "    %-*s | %-*s | %-*s\n", nameW, "Name", srcW, "Src", dstW, "Dst")
			fmt.Fprintf(&buf, "    %s-+-%s-+-%s\n", strings.Repeat("-", nameW), strings.Repeat("-", srcW), strings.Repeat("-", dstW))
			for _, id := range td.Indexes {
				srcStr := "-"
				dstStr := "-"
				if id.Src != nil {
					srcStr = formatIndexShort(id.Src)
				}
				if id.Dst != nil {
					dstStr = formatIndexShort(id.Dst)
				}
				fmt.Fprintf(&buf, "    %-*s | %-*s | %-*s\n", nameW, id.Name, srcW, srcStr, dstW, dstStr)
			}
		}

		// ForeignKeys
		if len(td.ForeignKeys) > 0 {
			buf.WriteString("  ForeignKeys:\n")
			nameW, srcW, dstW := calcFKWidths(td.ForeignKeys)
			fmt.Fprintf(&buf, "    %-*s | %-*s | %-*s\n", nameW, "Name", srcW, "Src", dstW, "Dst")
			fmt.Fprintf(&buf, "    %s-+-%s-+-%s\n", strings.Repeat("-", nameW), strings.Repeat("-", srcW), strings.Repeat("-", dstW))
			for _, fd := range td.ForeignKeys {
				srcStr := "-"
				dstStr := "-"
				if fd.Src != nil {
					srcStr = formatFKShort(fd.Src)
				}
				if fd.Dst != nil {
					dstStr = formatFKShort(fd.Dst)
				}
				fmt.Fprintf(&buf, "    %-*s | %-*s | %-*s\n", nameW, fd.Name, srcW, srcStr, dstW, dstStr)
			}
		}

		// PrimaryKey
		if td.PrimaryKey.Type != DiffTypeEqual {
			buf.WriteString("  PrimaryKey:\n")
			fmt.Fprintf(&buf, "    Src: [%s]\n", strings.Join(td.PrimaryKey.Src, ", "))
			fmt.Fprintf(&buf, "    Dst: [%s]\n", strings.Join(td.PrimaryKey.Dst, ", "))
		}

		if !td.HasDiff {
			buf.WriteString("  (no differences)\n")
		}

		buf.WriteString("\n")
	}

	fmt.Fprintf(&buf, "Summary: %d tables, %d differ, %d equal\n",
		result.Summary.TotalTables, result.Summary.DiffTables, result.Summary.EqualTables)

	return buf.Bytes(), nil
}

func calcColumnWidths(diffs []ColumnDiff) (nameW, srcW, dstW int) {
	nameW = 4 // "Name"
	srcW = 3  // "Src"
	dstW = 3  // "Dst"
	for _, cd := range diffs {
		if len(cd.Name) > nameW {
			nameW = len(cd.Name)
		}
		s := formatColumnShort(cd.Src)
		if len(s) > srcW {
			srcW = len(s)
		}
		d := formatColumnShort(cd.Dst)
		if len(d) > dstW {
			dstW = len(d)
		}
	}
	return
}

func calcIndexWidths(diffs []IndexDiff) (nameW, srcW, dstW int) {
	nameW = 4
	srcW = 3
	dstW = 3
	for _, id := range diffs {
		if len(id.Name) > nameW {
			nameW = len(id.Name)
		}
		s := formatIndexShort(id.Src)
		if len(s) > srcW {
			srcW = len(s)
		}
		d := formatIndexShort(id.Dst)
		if len(d) > dstW {
			dstW = len(d)
		}
	}
	return
}

func calcFKWidths(diffs []FKDiff) (nameW, srcW, dstW int) {
	nameW = 4
	srcW = 3
	dstW = 3
	for _, fd := range diffs {
		if len(fd.Name) > nameW {
			nameW = len(fd.Name)
		}
		s := formatFKShort(fd.Src)
		if len(s) > srcW {
			srcW = len(s)
		}
		d := formatFKShort(fd.Dst)
		if len(d) > dstW {
			dstW = len(d)
		}
	}
	return
}

func formatIndexShort(idx *Index) string {
	if idx == nil {
		return "-"
	}
	s := "(" + strings.Join(idx.Columns, ", ") + ")"
	if idx.IsUnique {
		s += " UNIQUE"
	}
	if idx.IsPrimary {
		s += " PRIMARY"
	}
	return s
}

func formatFKShort(fk *ForeignKey) string {
	if fk == nil {
		return "-"
	}
	return fk.Column + " → " + fk.RefTable + "(" + fk.RefColumn + ")"
}
