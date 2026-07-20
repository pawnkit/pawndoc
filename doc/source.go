package doc

import "strings"

func starts(text []byte) []int {
	result := []int{0}
	for i, value := range text {
		if value == '\n' {
			result = append(result, i+1)
		}
	}
	return result
}

func lineAt(starts []int, offset int) int {
	line := 1
	for i, start := range starts {
		if start > offset {
			break
		}
		line = i + 1
	}
	return line
}

func precedingComment(text []byte, offset int) string {
	if offset > len(text) {
		return ""
	}
	prefix := string(text[:offset])
	if lineStart := strings.LastIndex(prefix, "\n"); lineStart >= 0 {
		prefix = prefix[:lineStart+1]
	} else {
		prefix = ""
	}
	lines := strings.Split(prefix, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	var docs []string
	for len(lines) > 0 {
		line := strings.TrimSpace(lines[len(lines)-1])
		if !strings.HasPrefix(line, "///") {
			break
		}
		docs = append(docs, strings.TrimSpace(strings.TrimPrefix(line, "///")))
		lines = lines[:len(lines)-1]
	}
	if len(docs) > 0 {
		reverse(docs)
		return strings.Join(docs, "\n")
	}
	end := strings.LastIndex(prefix, "*/")
	if end < 0 || strings.TrimSpace(prefix[end+2:]) != "" {
		return ""
	}
	start := strings.LastIndex(prefix[:end], "/*")
	if start < 0 || start+3 > len(prefix) || prefix[start:start+3] != "/**" {
		return ""
	}
	body := prefix[start+3 : end]
	parts := strings.Split(body, "\n")
	for i := range parts {
		parts[i] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(parts[i]), "*"))
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func reverse(values []string) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
