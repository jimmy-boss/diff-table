// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 15:26
//
// --------------------------------------------
package difftable

import "encoding/json"

// JSONFormatter JSON 格式输出
type JSONFormatter struct{}

func (f *JSONFormatter) Format(result *CompareResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}
