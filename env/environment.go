// Package env implements functions for turning config values that are
// parameterized by the environment string or environment variables into
// unambiguous, concrete files, paths, and packages.
package env

import (
	"regexp"
	"strings"
)

// An Environment is a sequence of fields specified by an environment string.
type Environment []string

// NewEnvironment returns an Environment from an environment string `env`. The
// environment string has fields separated by spaces.
func NewEnvironment(env string) Environment {
	return strings.Fields(env)
}

// ValidateKey will determine if a key can be split into valid regexp fields.
func ValidateKey(key string) bool {
	pattern := newPattern(key)
	return allCompile(pattern.good) && allCompile(pattern.bad)
}

func allCompile(fields []string) bool {
	for _, field := range fields {
		_, err := regexp.Compile(field)
		if err != nil {
			return false
		}
	}
	return true
}

// Select determines which key in a slice of strings `keys` matches the
// environment. A key is a string of regular expressions that may match any
// field in the environment separated by spaces. The first key that matches the
// environment will be returned, or an empty string otherwise. The fields that
// matched the key in the order that they were matched are also returned as a
// slice.
//
// Note that it's possible for the empty returned string to actually be the
// matched key itself. This is because an entirely whitespace key acts as a
// wildcard that matches if no other non-wildcard keys match.
func (env Environment) Select(keys []string) (key string, fields []string) {
	for _, k := range keys {
		pattern := newPattern(k)
		if pattern.wildcard() {
			// Wildcards are fallbacks.
			key = k
		} else {
			fields = env.patternFields(pattern)
			if fields != nil {
				key = k
				return
			}
		}
	}
	return
}

// patternFields returns a slice of the feilds in the environment that match the
// good part of the passed pattern `p` in the order that they match in the
// pattern. If the pattern does not match the environment, nil is returned.
func (env Environment) patternFields(p pattern) []string {
	matches := env.matchingFields(p.good)
	if matches == nil {
		return nil
	}

	badMatches := env.matchingFields(p.bad)
	if badMatches != nil && len(badMatches) > 0 {
		return nil
	}

	return matches
}

// matchingFields returns the list of matching fields in the environment as they
// matched in order. If the environment does not match the pattern (i.e., there
// are fields that were not matched), nil is returned. An empty slice returns an
// empty slice.
func (env Environment) matchingFields(fields []string) []string {
	matches := make([]string, 0, len(fields))

	for _, field := range fields {
		f, err := regexp.Compile("^(?:" + field + ")$")
		if err != nil {
			return nil
		}
		matched := false
		for _, e := range env {
			if f.MatchString(e) {
				matches = append(matches, e)
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}
	}

	return matches
}

// A Match is the result of an environment matching a key. The subgroups in the
// match are used to replace the contents of other strings.
type Match struct {
	fields string
	regexp *regexp.Regexp
}

// NewMatch builds a Match from the key and fields returned from the Select
// method run from an Environment. Since the returned values should be from the
// Select method which returns already valid arguments, an invalid key string
// will panic.
func NewMatch(key string, fields []string) Match {
	fieldString := strings.Join(fields, " ")
	r := strings.Join(newPattern(key).good, " ")
	r = "^(?:" + r + ")$"
	regexp := regexp.MustCompile(r)

	return Match{fieldString, regexp}
}

// Replace will substitute all of the subgroup matches syntax in the string s.
func (m Match) Replace(s string) string {
	return m.regexp.ReplaceAllString(m.fields, s)
}

// A pattern is a collection of the patterns that the fields in the environment
// string should or should not pass.
type pattern struct {
	good []string
	bad  []string
}

var loneExclamation = regexp.MustCompile("(^| )!($| )")

// newPattern returns a pattern struct given an environment pattern string.
func newPattern(s string) pattern {
	parts := loneExclamation.Split(s, 2)

	good := strings.Fields(parts[0])
	bad := []string{}
	if len(parts) == 2 {
		badPart := loneExclamation.ReplaceAllLiteralString(parts[1], " ")
		bad = strings.Fields(badPart)
	}
	return pattern{good, bad}
}

// wildcard will return true if the pattern was empty (entirely whitespace) and
// thus a wildcard pattern.
func (p pattern) wildcard() bool {
	return len(p.good) == 0 && len(p.bad) == 0
}
