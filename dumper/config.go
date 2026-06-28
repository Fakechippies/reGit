package dumper

import "strings"

var unsafeConfigKeys = []string{
	"fsmonitor",
	"sshcommand",
	"askpass",
	"editor",
	"pager",
}

func sanitizeGitConfig(data string) string {
	lines := strings.Split(data, "\n")
	for i, line := range lines {
		if isUnsafeConfigLine(line) {
			lines[i] = "# " + line
		}
	}
	return strings.Join(lines, "\n")
}

func isUnsafeConfigLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "[") {
		return false
	}

	lower := strings.ToLower(strings.ReplaceAll(trimmed, " ", ""))
	for _, key := range unsafeConfigKeys {
		if strings.HasPrefix(lower, key) && strings.Contains(lower, "=") {
			return true
		}
		if strings.Contains(lower, "."+key+"=") {
			return true
		}
	}
	return false
}
