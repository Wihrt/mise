package tdd

import (
	"regexp"
	"strings"
)

var (
	commandRe = regexp.MustCompile(`Commande` + space + `*:` + space + `*(.*)$`)
	testRefRe = regexp.MustCompile("\\*\\*Test\\*\\*[^`]*`([^`]+)`")
	wiredCmd  = regexp.MustCompile("Commande" + space + "*:" + space + "*`mise run ([" + word + ":.\\-]+)")
)

// SectionIDs lists, in order, the IDs of the `## slug` section of TESTS.md.
func SectionIDs(text, slug string) []string {
	var ids []string
	seen := map[string]bool{}
	inside := false
	for _, line := range Lines(text) {
		switch {
		case strings.HasPrefix(line, "## "):
			inside = field(line, 1) != "" && field(line, 1) == slug
		case inside && strings.HasPrefix(line, "### "):
			if f := field(line, 1); f != "" {
				id := strings.TrimRight(f, ":")
				if !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
		}
	}
	return ids
}

// SpecCriteria lists the IDs under the acceptance criteria heading of a spec.
func SpecCriteria(text string) []string {
	var ids []string
	inside := false
	for _, line := range Lines(text) {
		if strings.HasPrefix(line, "## ") {
			h := strings.TrimSpace(line[3:])
			inside = h == "Critères d'acceptation" || h == "Acceptance criteria"
		} else if inside {
			if id := headingID(line); id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// AllIDs is the set of normalized IDs of every section.
func AllIDs(text string) map[string]bool {
	found := map[string]bool{}
	for _, line := range Lines(text) {
		if id := headingID(line); id != "" {
			found[Norm(id)] = true
		}
	}
	return found
}

// IDLabels maps each normalized ID to its written form (CALC-001).
func IDLabels(text string) map[string]string {
	labels := map[string]string{}
	for _, line := range Lines(text) {
		if id := headingID(line); id != "" {
			if _, ok := labels[Norm(id)]; !ok {
				labels[Norm(id)] = id
			}
		}
	}
	return labels
}

// runsTask reports whether a Commande runs `mise run <task>` exactly.
func runsTask(command, task string) bool {
	needle := "mise run " + task
	for from := 0; ; {
		i := strings.Index(command[from:], needle)
		if i < 0 {
			return false
		}
		end := from + i + len(needle)
		if end >= len(command) {
			return true
		}
		r := []rune(command[end:])[0]
		if !isWord(r) && r != ':' && r != '.' && r != '-' {
			return true
		}
		from = end
	}
}

// OutOfScope maps a normalized ID to the Commande of its section when that
// Commande is not `mise run <task>`. IDs in keep are always in scope.
func OutOfScope(text, task string, keep map[string]bool) map[string]string {
	result := map[string]string{}
	command, has := "", false
	for _, line := range Lines(text) {
		m := commandRe.FindStringSubmatch(line)
		switch {
		case strings.HasPrefix(line, "## "):
			has = false
		case m != nil && !strings.HasPrefix(line, "### "):
			command, has = strings.TrimSpace(m[1]), true
		case has && !runsTask(command, task):
			if id := headingID(line); id != "" && !keep[Norm(id)] {
				if _, ok := result[Norm(id)]; !ok {
					result[Norm(id)] = command
				}
			}
		}
	}
	return result
}

// IncompleteHeadings lists the `###` headings that lack a criterion or a test line.
func IncompleteHeadings(text string) []string {
	var res []string
	h, has, s, t := "", false, false, false
	for _, line := range Lines(text) {
		if strings.HasPrefix(line, "### ") {
			if has && (!s || !t) {
				res = append(res, h)
			}
			h, has, s, t = line, true, false, false
		}
		if strings.Contains(line, "**Spec**") || strings.Contains(line, "**Critère**") {
			s = true
		}
		if strings.Contains(line, "**Test**") {
			t = true
		}
	}
	if has && (!s || !t) {
		res = append(res, h)
	}
	return res
}

// IncompleteEntries lists the IDs of a section whose criterion lacks Étant donné, Quand or Alors.
func IncompleteEntries(text, slug string) []string {
	var res []string
	inside, cur := false, ""
	seen := map[string]bool{}
	for _, line := range append(Lines(text), "## ") {
		if strings.HasPrefix(line, "#") {
			if cur != "" && len(seen) < 3 {
				res = append(res, cur)
			}
			cur, seen = "", map[string]bool{}
			if strings.HasPrefix(line, "## ") {
				inside = field(line, 1) == slug && slug != ""
			} else if inside {
				cur = field(line, 1)
			}
			continue
		}
		for _, k := range []string{"Étant donné", "Quand", "Alors"} {
			if strings.Contains(line, "**"+k+"**") {
				seen[k] = true
			}
		}
	}
	return res
}

// CitedTests lists the test files TESTS.md cites, in order, once each.
func CitedTests(text string) []string {
	var files []string
	seen := map[string]bool{}
	for _, line := range Lines(text) {
		if m := testRefRe.FindStringSubmatch(line); m != nil && !seen[m[1]] {
			seen[m[1]] = true
			files = append(files, m[1])
		}
	}
	return files
}

// SectionTests maps each slug to the test files its section cites.
func SectionTests(text string) map[string]map[string]bool {
	res := map[string]map[string]bool{}
	slug := ""
	for _, line := range Lines(text) {
		if strings.HasPrefix(line, "## ") {
			slug = field(line, 1)
			continue
		}
		if m := testRefRe.FindStringSubmatch(line); slug != "" && m != nil {
			if res[slug] == nil {
				res[slug] = map[string]bool{}
			}
			res[slug][m[1]] = true
		}
	}
	return res
}

// ReplacedIDs lists the IDs of the given slugs' sections in before that are gone from after.
func ReplacedIDs(before, after string, slugs []string) map[string]bool {
	kept := AllIDs(after)
	gone := map[string]bool{}
	for _, s := range slugs {
		for _, id := range SectionIDs(before, s) {
			if !kept[Norm(id)] {
				gone[id] = true
			}
		}
	}
	return gone
}

// EntryText is the body of the `### id` entry.
func EntryText(text, id string) string {
	var lines []string
	found := false
	for _, line := range Lines(text) {
		if strings.HasPrefix(line, "#") {
			found = strings.HasPrefix(line, "### ") && field(line, 1) == id
		} else if found {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func indent(line string) int {
	return len([]rune(line)) - len([]rune(strings.TrimLeftFunc(line, isSpace)))
}

// ClauseCount is the number of clauses of an entry's Alors: 1 for a one-line
// Alors, else the items of its sub-list.
func ClauseCount(entry string) int {
	lines := Lines(entry)
	for n, line := range lines {
		_, after, ok := strings.Cut(line, "**Alors**")
		if !ok {
			continue
		}
		if strings.TrimSpace(after) != "" {
			return 1
		}
		count := 0
		for _, sub := range lines[n+1:] {
			if indent(sub) <= indent(line) {
				break
			}
			if strings.HasPrefix(strings.TrimLeftFunc(sub, isSpace), "- ") {
				count++
			}
		}
		return count
	}
	return 0
}

// WiredCommand is the task of the `mise run` Commande of the slug's section, "" when none.
func WiredCommand(text, slug string) string {
	var section []string
	inside := false
	for _, line := range Lines(text) {
		if strings.HasPrefix(line, "## ") {
			if inside {
				break
			}
			rest, ok := strings.CutPrefix(line, "## "+slug)
			if ok && slug != "" {
				last, next := []rune(slug)[len([]rune(slug))-1], rune(0)
				if rest != "" {
					next = []rune(rest)[0]
				}
				inside = isWord(last) != (rest != "" && isWord(next))
			}
		}
		if inside {
			section = append(section, line)
		}
	}
	if m := wiredCmd.FindStringSubmatch(strings.Join(section, "\n")); m != nil {
		return m[1]
	}
	return ""
}
