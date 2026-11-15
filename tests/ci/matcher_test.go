package ci

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// workflowMatch applies patterns in order, including re-inclusions after an
// exclusion. The supported metacharacters are the ones used by these filters.
type compiledPattern struct {
	re       *regexp.Regexp
	negative bool
}

func compilePatterns(patterns []string) ([]compiledPattern, error) {
	var compiled []compiledPattern
	for _, pattern := range patterns {
		if strings.ContainsAny(pattern, `[]+\`) {
			return nil, fmt.Errorf("unsupported syntax in pattern %q", pattern)
		}
		negative := strings.HasPrefix(pattern, "!")
		if negative {
			pattern = pattern[1:]
		}
		var re strings.Builder
		re.WriteByte('^')
		for i := 0; i < len(pattern); i++ {
			switch pattern[i] {
			case '*':
				if i+1 < len(pattern) && pattern[i+1] == '*' {
					i++
					if i+1 < len(pattern) && pattern[i+1] == '/' {
						i++
						re.WriteString("(.*/)?")
					} else {
						re.WriteString(".*")
					}
				} else {
					re.WriteString("[^/]*")
				}
			case '?':
				re.WriteString("[^/]")
			default:
				r, size := utf8.DecodeRuneInString(pattern[i:])
				re.WriteString(regexp.QuoteMeta(string(r)))
				i += size - 1
			}
		}
		re.WriteByte('$')
		matcher, err := regexp.Compile(re.String())
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
		compiled = append(compiled, compiledPattern{matcher, negative})
	}
	return compiled, nil
}

func matches(patterns []compiledPattern, path string) bool {
	matched := false
	for _, pattern := range patterns {
		if pattern.re.MatchString(path) {
			matched = !pattern.negative
		}
	}
	return matched
}

func workflowMatch(patterns []string, path string) (bool, error) {
	compiled, err := compilePatterns(patterns)
	return matches(compiled, path), err
}

type eventFilter struct {
	Paths       []string `yaml:"paths"`
	PathsIgnore []string `yaml:"paths-ignore"`
}

func (e eventFilter) includes(path string) (bool, error) {
	if len(e.Paths) > 0 && len(e.PathsIgnore) > 0 {
		return false, fmt.Errorf("paths and paths-ignore cannot coexist")
	}
	if len(e.Paths) > 0 {
		return workflowMatch(e.Paths, path)
	}
	if len(e.PathsIgnore) > 0 {
		ignored, err := workflowMatch(e.PathsIgnore, path)
		return !ignored, err
	}
	return true, nil
}

func TestWorkflowMatcher(t *testing.T) {
	tests := []struct {
		name     string
		filter   eventFilter
		path     string
		included bool
	}{
		{"star", eventFilter{Paths: []string{"a/*.go"}}, "a/file.go", true},
		{"star stops at slash", eventFilter{Paths: []string{"a/*.go"}}, "a/sub/file.go", false},
		{"globstar nested", eventFilter{Paths: []string{"a/**"}}, "a/sub/file.go", true},
		{"globstar zero directories", eventFilter{Paths: []string{"a/**/file.go"}}, "a/file.go", true},
		{"globstar more directories", eventFilter{Paths: []string{"a/**/file.go"}}, "a/b/c/file.go", true},
		{"leading globstar root", eventFilter{Paths: []string{"**/file.go"}}, "file.go", true},
		{"leading globstar nested", eventFilter{Paths: []string{"**/file.go"}}, "a/b/file.go", true},
		{"leading globstar suffix", eventFilter{Paths: []string{"**/file.go"}}, "file.go.txt", false},
		{"go module", eventFilter{Paths: []string{"go.*"}}, "go.mod", true},
		{"go checksum", eventFilter{Paths: []string{"go.*"}}, "go.sum", true},
		{"go module nested", eventFilter{Paths: []string{"go.*"}}, "vendor/go.mod", false},
		{"question mark", eventFilter{Paths: []string{"a/?.go"}}, "a/x.go", true},
		{"unicode literal and wildcard", eventFilter{Paths: []string{"文件/?.go"}}, "文件/測.go", true},
		{"question mark stops at slash", eventFilter{Paths: []string{"a?b"}}, "a/b", false},
		{"exclude", eventFilter{Paths: []string{"a/**", "!a/private/**"}}, "a/private/x", false},
		{"last inclusion wins", eventFilter{Paths: []string{"a/**", "!a/private/**", "a/private/keep"}}, "a/private/keep", true},
		{"last exclusion wins", eventFilter{Paths: []string{"a/private/keep", "!a/**"}}, "a/private/keep", false},
		{"ignore matches", eventFilter{PathsIgnore: []string{"docs/**"}}, "docs/x", false},
		{"ignore does not match", eventFilter{PathsIgnore: []string{"docs/**"}}, "pkg/x", true},
		{"no filter", eventFilter{}, "pkg/x", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.filter.includes(tt.path)
			if err != nil || got != tt.included {
				t.Fatalf("includes(%q) = %v, %v; want %v", tt.path, got, err, tt.included)
			}
		})
	}
	if _, err := (eventFilter{Paths: []string{"**"}, PathsIgnore: []string{"x"}}).includes("x"); err == nil {
		t.Fatal("paths and paths-ignore together must be rejected")
	}
	for _, pattern := range []string{"a/[b", "a/]b", "a/+b", `a/\b`} {
		if _, err := compilePatterns([]string{pattern}); err == nil || !strings.Contains(err.Error(), strconv.Quote(pattern)) {
			t.Errorf("unsupported pattern %q: got error %v", pattern, err)
		}
	}
}

func TestYAMLOnIsString(t *testing.T) {
	var doc map[string]any
	if err := yaml.Unmarshal([]byte("on:\n  pull_request:\n    paths: ['x/**']\n"), &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["on"]; !ok {
		t.Fatalf("yaml.v3 did not decode on as a string key: %#v", doc)
	}
}
