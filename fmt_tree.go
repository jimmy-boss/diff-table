// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 15:40
//
// --------------------------------------------
package difftable

import (
	"bytes"
	"fmt"
	"strings"
)

// TreeFormatter 树形缩进输出
type TreeFormatter struct{}

func (f *TreeFormatter) Format(result *CompareResult) ([]byte, error) {
	var buf bytes.Buffer

	for i, td := range result.Tables {
		if i > 0 {
			buf.WriteString("\n")
		}

		status := ""
		if td.HasDiff {
			status = " [DIFF]"
		} else {
			status = " [OK]"
		}
		fmt.Fprintf(&buf, "Table: %s%s\n", td.TableName, status)

		// Columns
		if len(td.Columns) > 0 {
			buf.WriteString("├── Columns\n")
			for j, cd := range td.Columns {
				prefix := "│   ├── "
				if j == len(td.Columns)-1 {
					prefix = "│   └── "
				}
				switch cd.Type {
				case DiffTypeEqual:
					fmt.Fprintf(&buf, "%s%s: %s\n", prefix, cd.Name, formatColumnShort(cd.Src))
				case DiffTypeAdded:
					fmt.Fprintf(&buf, "%s[+] %s: %s\n", prefix, cd.Name, formatColumnShort(cd.Dst))
				case DiffTypeRemoved:
					fmt.Fprintf(&buf, "%s[-] %s: %s\n", prefix, cd.Name, formatColumnShort(cd.Src))
				case DiffTypeChanged:
					fmt.Fprintf(&buf, "%s[~] %s: %s → %s\n", prefix, cd.Name, formatColumnShort(cd.Src), formatColumnShort(cd.Dst))
					for _, detail := range cd.Details {
						fmt.Fprintf(&buf, "│       %s: %v → %v\n", detail.Field, detail.Src, detail.Dst)
					}
				}
			}
		}

		// Indexes
		if len(td.Indexes) > 0 {
			buf.WriteString("├── Indexes\n")
			for j, id := range td.Indexes {
				prefix := "│   ├── "
				if j == len(td.Indexes)-1 {
					prefix = "│   └── "
				}
				switch id.Type {
				case DiffTypeEqual:
					fmt.Fprintf(&buf, "%s%s: %s\n", prefix, id.Name, formatIndexShort(id.Src))
				case DiffTypeAdded:
					fmt.Fprintf(&buf, "%s[+] %s: %s\n", prefix, id.Name, formatIndexShort(id.Dst))
				case DiffTypeRemoved:
					fmt.Fprintf(&buf, "%s[-] %s: %s\n", prefix, id.Name, formatIndexShort(id.Src))
				case DiffTypeChanged:
					fmt.Fprintf(&buf, "%s[~] %s: %s → %s\n", prefix, id.Name, formatIndexShort(id.Src), formatIndexShort(id.Dst))
				}
			}
		}

		// ForeignKeys
		if len(td.ForeignKeys) > 0 {
			buf.WriteString("├── ForeignKeys\n")
			for j, fd := range td.ForeignKeys {
				prefix := "│   ├── "
				if j == len(td.ForeignKeys)-1 {
					prefix = "│   └── "
				}
				switch fd.Type {
				case DiffTypeEqual:
					fmt.Fprintf(&buf, "%s%s: %s\n", prefix, fd.Name, formatFKShort(fd.Src))
				case DiffTypeAdded:
					fmt.Fprintf(&buf, "%s[+] %s: %s\n", prefix, fd.Name, formatFKShort(fd.Dst))
				case DiffTypeRemoved:
					fmt.Fprintf(&buf, "%s[-] %s: %s\n", prefix, fd.Name, formatFKShort(fd.Src))
				case DiffTypeChanged:
					fmt.Fprintf(&buf, "%s[~] %s\n", prefix, fd.Name)
				}
			}
		}

		// PrimaryKey
		pkPrefix := "└── "
		if td.PrimaryKey.Type != DiffTypeEqual {
			fmt.Fprintf(&buf, "%sPrimaryKey: [%s] → [%s]\n", pkPrefix,
				strings.Join(td.PrimaryKey.Src, ", "),
				strings.Join(td.PrimaryKey.Dst, ", "))
		} else if len(td.PrimaryKey.Src) > 0 {
			fmt.Fprintf(&buf, "%sPrimaryKey: [%s]\n", pkPrefix,
				strings.Join(td.PrimaryKey.Src, ", "))
		}
	}

	// Summary
	fmt.Fprintf(&buf, "\nSummary: %d tables, %d differ, %d equal\n",
		result.Summary.TotalTables, result.Summary.DiffTables, result.Summary.EqualTables)

	return buf.Bytes(), nil
}
