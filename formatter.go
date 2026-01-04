// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 15:25
//
// --------------------------------------------
package difftable

// Formatter 输出格式化接口
type Formatter interface {
	Format(result *CompareResult) ([]byte, error)
}

// getFormatter 根据 OutputFormat 返回对应的 Formatter
func getFormatter(outputFmt OutputFormat) Formatter {
	switch outputFmt {
	case OutputConsoleDiff:
		return &DiffFormatter{}
	case OutputConsoleTable:
		return &TableFormatter{}
	case OutputConsoleTree:
		return &TreeFormatter{}
	default:
		return &JSONFormatter{}
	}
}

// Output 输出结果（根据 outputFmt 格式化）
func (d *Differ) Output(result *CompareResult) ([]byte, error) {
	f := getFormatter(d.outputFmt)
	return f.Format(result)
}
