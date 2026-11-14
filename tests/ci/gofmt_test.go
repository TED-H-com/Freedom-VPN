package ci

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type gofmtSet struct {
	command string
	form    string
	files   []string
}

// Git's pathspec rules differ from shell globs, so let Git resolve them.
func trackedFiles(root string, pathspecs ...string) ([]string, error) {
	topCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	topCmd.Dir = root
	topOutput, err := topCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gofmt inventory needs a git checkout at %q: git rev-parse: %w", root, err)
	}
	top := strings.TrimSpace(string(topOutput))
	canonical := func(name string) (string, error) {
		absolute, err := filepath.Abs(name)
		if err != nil {
			return "", err
		}
		return filepath.EvalSymlinks(absolute)
	}
	actualRoot, err := canonical(root)
	if err != nil {
		return nil, fmt.Errorf("gofmt inventory needs a git checkout at %q (git top level %q): %w", root, top, err)
	}
	actualTop, err := canonical(top)
	if err != nil {
		return nil, fmt.Errorf("gofmt inventory needs a git checkout at %q (git top level %q): %w", root, top, err)
	}
	if actualRoot != actualTop {
		return nil, fmt.Errorf("gofmt inventory needs a git checkout at %q (git top level %q)", root, top)
	}
	args := append([]string{"ls-files", "--cached", "-z", "--"}, pathspecs...)
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gofmt inventory needs a git checkout: git ls-files: %w", err)
	}
	var files []string
	for _, file := range strings.Split(string(out), "\x00") {
		if file != "" {
			files = append(files, file)
		}
	}
	return files, nil
}

// substitutionEnd finds the end of a shell $(...) without evaluating it.
func substitutionEnd(text string, start int) (int, error) {
	var quote byte
	for i := start + 2; i < len(text); i++ {
		c := text[i]
		if quote == '\'' {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\\' {
			i++
			continue
		}
		if c == '$' && i+1 < len(text) && text[i+1] == '(' {
			end, err := substitutionEnd(text, i)
			if err != nil {
				return 0, err
			}
			i = end
			continue
		}
		if quote == '"' {
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
		case ')':
			return i, nil
		}
	}
	return 0, fmt.Errorf("unclosed shell command substitution")
}

type shellWord struct {
	value  string
	quoted bool
}

// shellWords accepts literal words and whole command substitutions only. It
// stops at shell operators; unsupported expansions are errors, not guesses.
func shellWords(text string) ([]shellWord, int, error) {
	var words []shellWord
	for i := 0; i < len(text); {
		if text[i] == ' ' || text[i] == '\t' || text[i] == '\r' {
			i++
			continue
		}
		if strings.ContainsRune("\n;|&<>(){}", rune(text[i])) {
			return words, i, nil
		}
		var word strings.Builder
		quoted := false
		for i < len(text) && !strings.ContainsRune(" \t\r\n;|&<>(){}", rune(text[i])) {
			c := text[i]
			switch {
			case c == '\'' || c == '"':
				quote := c
				quoted = true
				i++
				for i < len(text) && text[i] != quote {
					if quote == '"' && strings.ContainsRune("$`\\", rune(text[i])) {
						return nil, i, fmt.Errorf("unsupported quoted expansion")
					}
					word.WriteByte(text[i])
					i++
				}
				if i == len(text) {
					return nil, i, fmt.Errorf("unclosed quote")
				}
				i++
			case c == '$' && i+1 < len(text) && text[i+1] == '(':
				if word.Len() != 0 {
					return nil, i, fmt.Errorf("mixed command substitution")
				}
				end, err := substitutionEnd(text, i)
				if err != nil {
					return nil, i, err
				}
				word.WriteString(text[i : end+1])
				i = end + 1
				if i < len(text) && !strings.ContainsRune(" \t\r\n;|&<>(){}", rune(text[i])) {
					return nil, i, fmt.Errorf("mixed command substitution")
				}
			case strings.ContainsRune("$`\\", rune(c)):
				return nil, i, fmt.Errorf("unsupported shell expansion")
			default:
				word.WriteByte(c)
				i++
			}
		}
		words = append(words, shellWord{word.String(), quoted})
	}
	return words, len(text), nil
}

// Search the shell text and its command substitutions, but not literal quoted
// messages such as echo 'gofmt required'. No workflow command is executed.
func gofmtCommands(text string) ([]string, error) {
	text = strings.ReplaceAll(text, "\\\n", " ")
	var commands []string
	var scan func(string) error
	scan = func(text string) error {
		var quote byte
		for i := 0; i < len(text); i++ {
			c := text[i]
			if quote == '\'' {
				if c == quote {
					quote = 0
				}
				continue
			}
			if c == '\\' {
				i++
				continue
			}
			if c == '$' && i+1 < len(text) && text[i+1] == '(' {
				end, err := substitutionEnd(text, i)
				if err != nil {
					return err
				}
				if err := scan(text[i+2 : end]); err != nil {
					return err
				}
				i = end
				continue
			}
			if quote == '"' {
				if c == quote {
					quote = 0
				}
				continue
			}
			if c == '\'' || c == '"' {
				quote = c
				continue
			}
			if !strings.HasPrefix(text[i:], "gofmt") || (i > 0 && !strings.ContainsRune(" \t\r\n;|&({", rune(text[i-1]))) || (i+5 < len(text) && !strings.ContainsRune(" \t\r\n;|&)>", rune(text[i+5]))) {
				continue
			}
			_, end, err := shellWords(text[i:])
			if err != nil {
				return fmt.Errorf("gofmt command %q: %w", text[i:], err)
			}
			commands = append(commands, strings.TrimSpace(text[i:i+end]))
			i += end - 1
		}
		return nil
	}
	err := scan(text)
	return commands, err
}

func repoPath(value string) (string, error) {
	clean := path.Clean(value)
	if path.IsAbs(value) || clean == ".." || strings.HasPrefix(clean, "../") || strings.ContainsAny(value, "?[]{}") {
		return "", fmt.Errorf("unsupported path %q", value)
	}
	return clean, nil
}

func gofmtInputs(root, text string, tracked []string) ([]gofmtSet, error) {
	commands, err := gofmtCommands(text)
	if err != nil {
		return nil, err
	}
	var sets []gofmtSet
	for _, command := range commands {
		set, err := resolveGofmt(root, command, tracked)
		if err != nil {
			return nil, fmt.Errorf("gofmt command %q: %w", command, err)
		}
		sets = append(sets, set)
	}
	return sets, nil
}

func resolveGofmt(root, command string, tracked []string) (gofmtSet, error) {
	result := gofmtSet{command: command}
	words, _, err := shellWords(command)
	if err != nil {
		return result, err
	}
	if len(words) < 3 || words[0].value != "gofmt" || words[1].value != "-l" {
		return result, fmt.Errorf("expected gofmt -l with file arguments")
	}
	files := map[string]bool{}
	if strings.HasPrefix(words[2].value, "$(") {
		if len(words) != 3 || words[2].quoted {
			return result, fmt.Errorf("unsupported command substitution arguments")
		}
		inner := strings.TrimSuffix(strings.TrimPrefix(words[2].value, "$("), ")")
		args, end, err := shellWords(inner)
		if err != nil || end != len(inner) || len(args) < 2 {
			return result, fmt.Errorf("unsupported file-list command %q", inner)
		}
		switch {
		case len(args) >= 6 && args[0].value == "git" && args[1].value == "ls-files" && args[2].value == "--cached" && args[3].value == "--others" && args[4].value == "--exclude-standard":
			result.form = "git ls-files"
			var specs []string
			for _, arg := range args[5:] {
				if strings.HasPrefix(arg.value, "-") || (!arg.quoted && strings.ContainsAny(arg.value, "*?[")) {
					return result, fmt.Errorf("unsupported git pathspec %q", arg.value)
				}
				specs = append(specs, arg.value)
			}
			listed, err := trackedFiles(root, specs...)
			if err != nil {
				return result, err
			}
			for _, file := range listed {
				files[file] = true
			}
		case len(args) >= 5 && args[0].value == "rg" && args[1].value == "--files" && args[len(args)-2].value == "-g" && args[len(args)-1].value == "*.go" && args[len(args)-1].quoted:
			result.form = "rg --files"
			for _, arg := range args[2 : len(args)-2] {
				dir, err := repoPath(arg.value)
				if err != nil || strings.Contains(dir, "*") || strings.HasPrefix(dir, "-") {
					return result, fmt.Errorf("unsupported rg directory %q", arg.value)
				}
				info, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir)))
				if err != nil || !info.IsDir() {
					return result, fmt.Errorf("rg directory %q does not exist", dir)
				}
				for _, file := range tracked {
					if (dir == "." || strings.HasPrefix(file, dir+"/")) && strings.HasSuffix(file, ".go") {
						files[file] = true
					}
				}
			}
		default:
			return result, fmt.Errorf("unsupported file-list command %q", inner)
		}
	} else {
		result.form = "paths"
		for _, arg := range words[2:] {
			name, err := repoPath(arg.value)
			if err != nil || strings.HasPrefix(name, "-") {
				return result, fmt.Errorf("unsupported file argument %q", arg.value)
			}
			if strings.Contains(name, "*") && arg.quoted {
				return result, fmt.Errorf("quoted glob %q is not a shell expansion", name)
			}
			if !strings.Contains(name, "*") {
				info, err := os.Stat(filepath.Join(root, filepath.FromSlash(name)))
				if err != nil {
					return result, fmt.Errorf("file argument %q: %w", name, err)
				}
				for _, file := range tracked {
					if file == name || (info.IsDir() && (name == "." || strings.HasPrefix(file, name+"/")) && strings.HasSuffix(file, ".go")) {
						files[file] = true
					}
				}
				continue
			}
			matched := false
			for _, file := range tracked {
				for candidate := file; candidate != "."; candidate = path.Dir(candidate) {
					ok, err := path.Match(name, candidate)
					if err != nil {
						return result, err
					}
					if ok {
						matched = true
						if candidate == file || strings.HasSuffix(file, ".go") {
							files[file] = true
						}
					}
				}
			}
			if !matched {
				return result, fmt.Errorf("shell glob %q matches no tracked path", name)
			}
		}
	}
	result.files = sortedKeys(files)
	return result, nil
}
