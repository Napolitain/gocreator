package services

import "path/filepath"

func testPath(parts ...string) string {
	allParts := make([]string, 0, len(parts)+1)
	allParts = append(allParts, string(filepath.Separator))
	allParts = append(allParts, parts...)
	return filepath.Join(allParts...)
}
