package deploy

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aus-hawk/estragon/config"
)

type mockFileDeployer struct {
	expectedMap    map[string]string
	expectedMethod string
	t              *testing.T
}

func (d mockFileDeployer) testExpectedMap(m map[string]string) error {
	if d.expectedMap == nil {
		return errors.New(d.expectedMethod + " error")
	}

	if !reflect.DeepEqual(d.expectedMap, m) {
		d.t.Errorf(
			"expected resulting file map to be %#v, got %#v",
			d.expectedMap,
			m,
		)
	}
	return nil
}

func (d mockFileDeployer) Copy(m map[string]string, dot string) error {
	if d.expectedMethod != "copy" {
		d.t.Error("copy called when it shouldn't have")
	}
	return d.testExpectedMap(m)
}

func (d mockFileDeployer) Symlink(m map[string]string, dot string) error {
	if d.expectedMethod != "symlink" {
		d.t.Error("symlink called when it shouldn't have")
	}
	return d.testExpectedMap(m)
}

func (d mockFileDeployer) Expand(s string, dot string) string {
	return s
}

type mockDotManager struct {
	shouldErr bool
	method    string
	root      string
	dotPrefix bool
	rules     map[string]string
	deploy    [][]string
}

func (d mockDotManager) DotConfig(string) config.DotConfig {
	return config.DotConfig{
		Method:    d.method,
		Root:      d.root,
		DotPrefix: d.dotPrefix,
		Rules:     d.rules,
		Deploy:    d.deploy,
	}
}

func TestNewDotfileDeployer(t *testing.T) {
	expected := DotfileDeployer{mockDotManager{}, mockFileDeployer{}, "", nil}
	actual := NewDotfileDeployer(mockDotManager{}, mockFileDeployer{}, "", nil)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %#v, got %#v", expected, actual)
	}
}

func TestDeploy(t *testing.T) {
	f := filepath.Join // used a lot, just a shorthand
	tests := []struct {
		desc           string
		mgr            mockDotManager
		dotRoot        string
		files          []string
		deployRun      CmdRunner
		expectedMap    map[string]string
		expectedMethod string
		err            string
	}{
		{
			"Bad method",
			mockDotManager{
				method: "bad-method",
			},
			"/dot/root",
			[]string{"doesnt", "matter"},
			nil,
			nil,
			"",
			"bad-method is not a valid method",
		},
		{
			`Method "none" does nothing`,
			mockDotManager{
				method: "none",
			},
			"/dot/root",
			[]string{"doesnt", "matter"},
			nil,
			nil,
			"",
			"",
		},
		{
			"symlink error on shallow",
			mockDotManager{
				method: "shallow",
			},
			"/dot/root",
			[]string{"doesnt", "matter"},
			nil,
			nil,
			"symlink",
			"symlink error",
		},
		{
			"symlink error on deep",
			mockDotManager{
				method: "deep",
			},
			"/dot/root",
			[]string{"doesnt", "matter"},
			nil,
			nil,
			"symlink",
			"symlink error",
		},
		{
			"copy error on copy",
			mockDotManager{
				method: "copy",
			},
			"/dot/root",
			[]string{"doesnt", "matter"},
			nil,
			nil,
			"copy",
			"copy error",
		},
		{
			`Method "shallow" no rules`,
			mockDotManager{
				method: "shallow",
				root:   "/out/root",
			},
			"/dot/root",
			[]string{"doesnt", "matter"},
			nil,
			map[string]string{
				f("/dot/root", "dotname"): "/out/root",
			},
			"symlink",
			"",
		},
		{
			`Method "shallow" with rules uses map of valid files`,
			mockDotManager{
				method:    "shallow",
				root:      "/out/root",
				dotPrefix: true, // should be ignored anyway
				rules: map[string]string{
					"a":     "/link/a/dot",
					"c":     "/should/be/ignored",
					"d/f":   "/dir/loc",
					"d/f/g": "/nest/file",
				},
			},
			"/dot/root",
			[]string{"a", "b", "dot-c", "d/e", "d/f/g", "d/f/h"},
			nil,
			map[string]string{
				f("/dot/root", "dotname", "a"):     "/link/a/dot",
				f("/dot/root", "dotname", "d/f"):   "/dir/loc",
				f("/dot/root", "dotname", "d/f/g"): "/nest/file",
			},
			"symlink",
			"",
		},
		{
			`Method "deep" links to root unless a rule exists`,
			mockDotManager{
				method:    "deep",
				root:      "/out/root",
				dotPrefix: true,
				rules: map[string]string{
					"a":     "/link/a/dot",
					"c":     "/should/be/ignored",
					"d/f":   "/dir/loc",
					"d/f/g": "/nest/file",
				},
			},
			"/dot/root",
			[]string{"a", "b", "dot-c", "d/e", "d/f/g", "d/f/h"},
			nil,
			map[string]string{
				// bound by rules
				f("/dot/root", "dotname", "a"):     "/link/a/dot",
				f("/dot/root", "dotname", "d/f/g"): "/nest/file",
				f("/dot/root", "dotname", "d/f/h"): "/dir/loc/h",
				// bound by root
				f("/dot/root", "dotname", "b"):     f("/out/root", "b"),
				f("/dot/root", "dotname", "dot-c"): f("/out/root", ".c"),
				f("/dot/root", "dotname", "d/e"):   f("/out/root", "d/e"),
			},
			"symlink",
			"",
		},
		{
			`Method "copy" copies to root unless a rule exists`,
			mockDotManager{
				method:    "copy",
				root:      "/out/root",
				dotPrefix: true,
				rules: map[string]string{
					"a":     "/link/a/dot",
					"c":     "/should/be/ignored",
					"d/f":   "/dir/loc",
					"d/f/g": "/nest/file",
				},
			},
			"/dot/root",
			[]string{"a", "b", "dot-c", "d/e", "d/f/g", "d/f/h"},
			nil,
			map[string]string{
				// bound by rules
				f("/dot/root", "dotname", "a"):     "/link/a/dot",
				f("/dot/root", "dotname", "d/f/g"): "/nest/file",
				f("/dot/root", "dotname", "d/f/h"): "/dir/loc/h",
				// bound by root
				f("/dot/root", "dotname", "b"):     f("/out/root", "b"),
				f("/dot/root", "dotname", "dot-c"): f("/out/root", ".c"),
				f("/dot/root", "dotname", "d/e"):   f("/out/root", "d/e"),
			},
			"copy",
			"",
		},
		{
			"Successful deployment",
			mockDotManager{
				method: "none",
				deploy: [][]string{{"a", "b"}, {"c", "d"}},
			},
			"",
			nil,
			func([]string) (int, error) {
				return 0, nil
			},
			nil,
			"",
			"",
		},
		{
			"Bad deployment from non-zero code",
			mockDotManager{
				method: "none",
				deploy: [][]string{{"a", "b"}, {"c", "d"}},
			},
			"",
			nil,
			func([]string) (int, error) {
				return 1, nil
			},
			nil,
			"",
			"Command a b returned 1",
		},
		{
			"Bad deployment from bad run",
			mockDotManager{
				method: "none",
				deploy: [][]string{{"a", "b"}, {"c", "d"}},
			},
			"",
			nil,
			func([]string) (int, error) {
				return 0, errors.New("Uh oh")
			},
			nil,
			"",
			"Uh oh",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			deployer := DotfileDeployer{
				test.mgr,
				mockFileDeployer{
					expectedMap:    test.expectedMap,
					expectedMethod: test.expectedMethod,
					t:              t,
				},
				test.dotRoot,
				test.deployRun,
			}

			err := deployer.Deploy("dotname", test.files, false)

			if test.err != "" {
				if err == nil || err.Error() != test.err {
					t.Errorf(
						"expected err %s, got %#v",
						test.err,
						err,
					)
				}
			} else if err != nil {
				t.Errorf("expected no error, got %#v", err)
			}
		})
	}
}
