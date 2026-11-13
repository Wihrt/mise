package domain

import (
	"regexp"
	"strings"
)

var (
	plainWord = regexp.MustCompile(`^[A-Za-z][A-Za-z'-]*$`)
	// regexSigns are the characters that make a pattern a pattern, not a question.
	regexSigns = `\[](){}^$*+?.`
)

// Exploratory reports a search pattern that reads like a question about the
// code rather than like a string to find: at least three plain words in a
// row, or at least three plain alternatives.
func Exploratory(pattern string) bool {
	if strings.ContainsAny(pattern, regexSigns) {
		return false
	}
	return plainWords(strings.Fields(pattern)) || plainWords(strings.Split(pattern, "|"))
}

func plainWords(words []string) bool {
	if len(words) < 3 {
		return false
	}
	for _, w := range words {
		if !plainWord.MatchString(strings.TrimSpace(w)) {
			return false
		}
	}
	return true
}
