package dotfile

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewResolver(t *testing.T) {
	expected := Resolver{"dotRoot", "outRoot", true, nil}
	actual := NewResolver("dotRoot", "outRoot", true, nil)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %#v, got %#v", expected, actual)
	}
}

func goodExpand(s string) (string, error) {
	return s, nil
}

func badExpand(s string) (string, error) {
	return "", errors.New("Bad expansion")
}

func TestDeepResolve(t *testing.T) {
	resolver := Resolver{dotRoot: "/dot/root", outRoot: "/out/root"}
	dr := func(p string) string {
		return filepath.Join(resolver.dotRoot, p)
	}
	or := func(p string) string {
		return filepath.Join(resolver.outRoot, p)
	}

	tests := []struct {
		desc      string
		dotPrefix bool
		expand    PathExpander
		files     []string
		rules     map[string]string
		resolved  map[string]string
		shouldErr bool
	}{
		{
			"No files",
			false,
			goodExpand,
			nil,
			map[string]string{"a": "b", "c": "d"},
			map[string]string{},
			false,
		},
		{
			"No rules with no dot-prefix expansion",
			false,
			goodExpand,
			[]string{"a", "b/c", "d/e/f", "d/e/g", "dot-h", "dot-i/j"},
			nil,
			map[string]string{
				dr("a"):       or("a"),
				dr("b/c"):     or("b/c"),
				dr("d/e/f"):   or("d/e/f"),
				dr("d/e/g"):   or("d/e/g"),
				dr("dot-h"):   or("dot-h"),
				dr("dot-i/j"): or("dot-i/j"),
			},
			false,
		},
		{
			"No rules",
			true,
			goodExpand,
			[]string{"a", "b/c", "d/e/f", "d/e/g", "dot-h", "dot-i/j"},
			nil,
			map[string]string{
				dr("a"):       or("a"),
				dr("b/c"):     or("b/c"),
				dr("d/e/f"):   or("d/e/f"),
				dr("d/e/g"):   or("d/e/g"),
				dr("dot-h"):   or(".h"),
				dr("dot-i/j"): or(".i/j"),
			},
			false,
		},
		{
			"Rules with no dot-prefix expansion",
			false,
			goodExpand,
			[]string{"a", "b/dot-c", "d/dot-e/f", "d/dot-e/g", "dot-h"},
			map[string]string{
				"d": "/d/dir",
				"b": "/b/root",
				"a": "~/a-file.txt",
			},
			map[string]string{
				dr("a"):         "~/a-file.txt",
				dr("b/dot-c"):   "/b/root/dot-c",
				dr("d/dot-e/f"): "/d/dir/dot-e/f",
				dr("d/dot-e/g"): "/d/dir/dot-e/g",
				dr("dot-h"):     or("dot-h"),
			},
			false,
		},
		{
			"Rules with dot-prefix expansion",
			true,
			goodExpand,
			[]string{"a", "b/dot-c", "d/dot-e/f", "d/dot-e/g", "dot-h"},
			map[string]string{
				"d": "/d/dir",
				"b": "/b/root",
				"a": "~/a-file.txt",
			},
			map[string]string{
				dr("a"):         "~/a-file.txt",
				dr("b/dot-c"):   "/b/root/.c",
				dr("d/dot-e/f"): "/d/dir/.e/f",
				dr("d/dot-e/g"): "/d/dir/.e/g",
				dr("dot-h"):     or(".h"),
			},
			false,
		},
		{
			"Files that map to nothing are ignored",
			true,
			goodExpand,
			[]string{"ignored", "not/ignored"},
			map[string]string{"ignored": ""},
			map[string]string{dr("not/ignored"): or("not/ignored")},
			false,
		},
		{
			"Empty key indicates ruleless files are ignored",
			true,
			goodExpand,
			[]string{"ignored", "not/ignored"},
			map[string]string{"not/ignored": "~/here", "": ""},
			map[string]string{dr("not/ignored"): "~/here"},
			false,
		},
		{
			"Bad expansion",
			false,
			badExpand,
			[]string{"a"},
			nil,
			nil,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			resolver.dotPrefix = test.dotPrefix
			resolver.expand = test.expand
			resolved, err := resolver.DeepResolve(
				test.files,
				test.rules,
			)

			if err == nil && test.shouldErr {
				t.Fatal("Expected non-nil err")
			} else if err != nil && !test.shouldErr {
				t.Fatal("Expected nil err")
			} else if err != nil {
				return
			}

			if !reflect.DeepEqual(test.resolved, resolved) {
				t.Errorf(
					"Expected %#v, got %#v",
					test.resolved,
					resolved,
				)
			}
		})
	}
}

func TestShallowResolve(t *testing.T) {
	resolver := Resolver{dotRoot: "/dot/root", outRoot: "/out/root"}
	dr := func(p string) string {
		return filepath.Join(resolver.dotRoot, p)
	}

	tests := []struct {
		desc      string
		dotPrefix bool
		expand    PathExpander
		files     []string
		rules     map[string]string
		resolved  map[string]string
		shouldErr bool
	}{
		{
			"No files or rules",
			false,
			goodExpand,
			nil,
			nil,
			map[string]string{resolver.dotRoot: resolver.outRoot},
			false,
		},
		{
			"No files with rules",
			false,
			goodExpand,
			nil,
			map[string]string{"shouldnt": "matter"},
			map[string]string{resolver.dotRoot: resolver.outRoot},
			false,
		},
		{
			"No rules with files",
			false,
			goodExpand,
			[]string{"fileA", "file/b", "file/c"},
			nil,
			map[string]string{resolver.dotRoot: resolver.outRoot},
			false,
		},
		{
			"Files and rules",
			true,
			goodExpand,
			[]string{"ignored", "not/ignored"},
			map[string]string{
				"not/ignored":  "~/f.txt",
				"ignored/rule": "nowhere",
				"ignored":      "",
			},
			map[string]string{dr("not/ignored"): "~/f.txt"},
			false,
		},
		{
			"Bad expand",
			true,
			badExpand,
			nil,
			nil,
			nil,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			resolver.dotPrefix = test.dotPrefix
			resolver.expand = test.expand
			resolved, err := resolver.ShallowResolve(
				test.files,
				test.rules,
			)

			if err == nil && test.shouldErr {
				t.Fatal("Expected non-nil err")
			} else if err != nil && !test.shouldErr {
				t.Fatal("Expected nil err")
			} else if err != nil {
				return
			}

			if !reflect.DeepEqual(test.resolved, resolved) {
				t.Errorf(
					"Expected %#v, got %#v",
					test.resolved,
					resolved,
				)
			}
		})
	}
}
