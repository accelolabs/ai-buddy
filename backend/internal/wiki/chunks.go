package wiki

import (
	"strings"
)

// Split prefers paragraph boundaries and measures Unicode code points, including in code blocks.
func Split(s string) []string {
	r := []rune(s)
	parts := []string{}
	for len(r) > 0 {
		end := min(1200, len(r))
		if end < len(r) {
			for j := end; j > 600; j-- {
				if r[j-1] == '\n' && r[j-2] == '\n' {
					end = j
					break
				}
			}
		}
		part := strings.TrimSpace(string(r[:end]))
		if part != "" {
			parts = append(parts, part)
		}
		if end == len(r) {
			break
		}
		r = r[end-min(150, end):]
	}
	return parts
}
