package coverage

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseLCOV reads an LCOV report. File names are relative to the repository
// root. A branch whose count is "-" never ran: it counts as not taken.
func ParseLCOV(text string) (Measure, error) {
	m := Measure{}
	var file string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(l)
		if name, ok := strings.CutPrefix(l, "SF:"); ok {
			file = name
			if m[file] == nil {
				m[file] = map[span]bool{}
			}
			continue
		}
		if data, ok := strings.CutPrefix(l, "DA:"); ok {
			n, count, _ := strings.Cut(data, ",")
			ln, err1 := strconv.Atoi(n)
			c, err2 := strconv.Atoi(count)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("bad lcov line %q", l)
			}
			s := span{start: pos{line: ln}, end: pos{line: ln}, kind: line}
			m[file][s] = m[file][s] || c > 0
		}
		if data, ok := strings.CutPrefix(l, "BRDA:"); ok {
			i := strings.LastIndex(data, ",")
			n, id, _ := strings.Cut(data[:max(i, 0)], ",")
			ln, err := strconv.Atoi(n)
			if i < 0 || err != nil {
				return nil, fmt.Errorf("bad lcov line %q", l)
			}
			s := span{start: pos{line: ln}, end: pos{line: ln}, kind: branch, desc: id}
			c, err := strconv.Atoi(data[i+1:])
			m[file][s] = m[file][s] || (err == nil && c > 0)
		}
	}
	return m, nil
}
