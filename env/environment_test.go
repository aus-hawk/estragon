package env

import (
	"reflect"
	"regexp"
	"testing"
)

func TestNewEnvironment(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		out  Environment
	}{
		{"Empty env", "", Environment{}},
		{"Empty env with spaces", "   ", Environment{}},
		{"Single field in env", "wow", Environment{"wow"}},
		{"Multiple fields in env", "big test", Environment{"big", "test"}},
		{
			"Multiple fields with various characters",
			"distro driver:thing m0re $tuff#",
			Environment{"distro", "driver:thing", "m0re", "$tuff#"},
		},
		{
			"Many interspersed spaces between fields",
			"  a   whole   bunch   of spaces  ",
			Environment{"a", "whole", "bunch", "of", "spaces"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			out := NewEnvironment(test.in)
			if !reflect.DeepEqual(out, test.out) {
				t.Errorf("expected %#v, got %#v", test.out, out)
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		out  bool
	}{
		{"Empty field", "", true},
		{"Single field in key", "wow", true},
		{"Multiple fields in key", "big test", true},
		{"Good and bad fields in key", "test with ! bad fields", true},
		{
			"Invalid key due to spaces in regexp",
			"(this test)",
			false,
		},
		{
			"Invalid key due to spaces in bad regexp",
			"good side no problems ! bad[ ]key",
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			out := ValidateKey(test.in)
			if out != test.out {
				t.Errorf("expected %#v, got %#v", test.out, out)
			}
		})
	}
}

func TestEnvSelect(t *testing.T) {
	keys := []string{
		"",
		" this test",
		"test(.*)regexp",
		"mult.ple regex.",
		"many  regexp ! bad|worse",
	}
	tests := []struct {
		desc     string
		env      Environment
		outKey   string
		outMatch Match
	}{
		{
			"Select wildcard",
			Environment{"no", "match"},
			keys[0],
			Match{},
		},
		{
			`Select " this test"`,
			Environment{"useless", "this", "key", "test"},
			keys[1],
			Match{"this test", regexp.MustCompile("^(?:this test)$")},
		},
		{
			`Select "test(.*)regexp"`,
			Environment{"testabcdregexp", "key", "testefghregexp"},
			keys[2],
			Match{
				"testabcdregexp",
				regexp.MustCompile("^(?:test(.*)regexp)$"),
			},
		},
		{
			`Select "mult.ple regex."`,
			Environment{"regexx", "multuple", "abcdefg"},
			keys[3],
			Match{
				"multuple regexx",
				regexp.MustCompile("^(?:mult.ple regex.)$"),
			},
		},
		{
			`Select "multiple regexp ! bad|worse"`,
			Environment{"regexp", "many", "abcdefg"},
			keys[4],
			Match{
				"many regexp",
				regexp.MustCompile("^(?:many regexp)$"),
			},
		},
		{
			`Don't select "multiple regexp ! bad|worse" because bad`,
			Environment{"regexp", "many", "bad"},
			"",
			Match{},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			key, match := test.env.Select(keys)

			if test.outKey != key {
				t.Errorf(
					"expected key to be %#v, got %#v",
					test.outKey,
					key,
				)
			}

			if !reflect.DeepEqual(test.outMatch.fields, match.fields) {
				t.Errorf(
					"expected fields to be %#v, got %#v",
					test.outMatch.fields,
					match.fields,
				)
			}

			if (test.outMatch.regexp == nil) != (match.regexp == nil) {
				t.Fatalf(
					"expected regexp to be to be %#v, got %#v",
					test.outMatch.fields,
					match.fields,
				)
			}

			if match.regexp == nil {
				// Skip if no regexp.
				return
			}
			if test.outMatch.regexp.String() != match.regexp.String() {
				t.Errorf(
					"expected regexp to be %#v, got %#v",
					test.outMatch.regexp.String(),
					match.regexp.String(),
				)
			}
		})
	}
}

func TestMatchReplace(t *testing.T) {
	match := Match{
		"distro:arch driver:amd test:passed",
		regexp.MustCompile("^(?:distro:(.+) driver:(.+) test:passed)$"),
	}
	replaced := match.Replace("package-${2}-$1")
	expected := "package-amd-arch"
	if replaced != expected {
		t.Fatalf("expected %#v, got %#v", expected, replaced)
	}
}
