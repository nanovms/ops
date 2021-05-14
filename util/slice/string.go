package slice

import "strings"

// ExcludeWhitespaces removes whitespaces from given slice
func ExcludeWhitespaces(arr []string) []string {
	result := make([]string, 0)
	for _, h := range arr {
		if strings.Trim(h, " ") == "" {
			continue
		}
		result = append(result, h)
	}
	return result
}
