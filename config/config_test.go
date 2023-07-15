package config

import (
	"reflect"
	"testing"
)

const goodYaml = `
method: deep
root: xdg

check-cmd:
  distro-one: ["pkgmgr", "check"]
  distro-two: ["packy", "query"]
  "t..t": ["testpkg-check"]
install-cmd:
  distro-one: ["pkgmgr", "install"]
  distro-two: ["packy", "get"]
  "t..t": ["testpkg-install"]

validate:
  test:
    - "this and that"
    - "that plus this"
  tust:
    - "thus und thut"
    - "thut plus thus"

environments:
  test:
    method: shallow
    root: home
    dot-prefix: false

packages:
  foo:
    test(s?): ["bar$1", baz]

dots:
  dotfile:
    method: copy
    root: "/test/dir/:"
    rules:
      test:
        "dir/file": "/new/loc/file"
    packages:
      foo: "For tests"
  commonless:
    packages:
      quux: "I don't know but it sounds important"
  templated:
    rules:
      "template-(.*)":
        "file-$1": "/outfile"
  deployable:
    deploy:
      test:
        - ["command", "one"]
        - ["cmd", "$TWO"]
`

var goodSchema = schema{
	Common: common{
		Method: "deep",
		Root:   "xdg",
	},
	CheckCmd: map[string][]string{
		"distro-one": {"pkgmgr", "check"},
		"distro-two": {"packy", "query"},
		"t..t":       {"testpkg-check"},
	},
	InstallCmd: map[string][]string{
		"distro-one": {"pkgmgr", "install"},
		"distro-two": {"packy", "get"},
		"t..t":       {"testpkg-install"},
	},
	Validate: map[string][]string{
		"test": {
			"this and that",
			"that plus this",
		},
		"tust": {
			"thus und thut",
			"thut plus thus",
		},
	},
	Environments: map[string]common{
		"test": {
			Method:    "shallow",
			Root:      "home",
			DotPrefix: new(bool),
		},
	},
	Packages: map[string]map[string][]string{
		"foo": {
			"test(s?)": {"bar$1", "baz"},
		},
	},
	Dots: map[string]dot{
		"dotfile": {
			Common: common{
				Method: "copy",
				Root:   "/test/dir/:",
			},
			Rules: map[string]map[string]string{
				"test": {
					"dir/file": "/new/loc/file",
				},
			},
			Packages: map[string]string{
				"foo": "For tests",
			},
		},
		"commonless": {
			Packages: map[string]string{
				"quux": "I don't know but it sounds important",
			},
		},
		"templated": {
			Rules: map[string]map[string]string{
				"template-(.*)": {
					"file-$1": "/outfile",
				},
			},
		},
		"deployable": {
			Deploy: map[string][][]string{
				"test": {
					{"command", "one"},
					{"cmd", "$TWO"},
				},
			},
		},
	},
}

type mockEnvSelector struct {
	key    string
	fields []string
}

func (s mockEnvSelector) Select(keys []string) (key string, fields []string) {
	return s.key, s.fields
}

func (s mockEnvSelector) Matches(e string) bool {
	// Fail if a string contains 'x'
	for _, c := range e {
		if c == 'x' {
			return false
		}
	}
	return true
}

func TestNewConfig(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		c    Config
		err  bool
	}{
		{"Bad YAML", "]", Config{}, true},
		{
			"Good YAML",
			goodYaml,
			Config{
				schema:   goodSchema,
				selector: mockEnvSelector{},
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			c, err := NewConfig([]byte(test.in), mockEnvSelector{})

			if err != nil && !test.err {
				t.Fatal("expected err to be nil, was " + err.Error())
			} else if err == nil && test.err {
				t.Fatal("expected err to be non-nil, was nil")
			} else if err != nil {
				// Don't compare invalid configs.
				return
			}

			if !reflect.DeepEqual(test.c, c) {
				t.Errorf(
					"expected config to be %#v, got %#v",
					test.c,
					c,
				)
			}
		})
	}
}

func TestValidateEnv(t *testing.T) {
	tests := []struct {
		desc           string
		validateList   map[string][]string
		shouldValidate bool
	}{
		{
			"No validate",
			nil,
			true,
		},
		{
			"Empty validate",
			map[string][]string{},
			true,
		},
		{
			"Environment doesn't match any keys",
			map[string][]string{
				"badxenv": {"badxkey"},
				"anxther": {"bad env xnd key"},
			},
			true,
		},
		{
			"Keys that match env pass",
			map[string][]string{
				"good":         {"badxkey", "its ok, this is good"},
				"bax":          {"shouldn't match thisx"},
				"another good": {"yup", "should be fine"},
			},
			true,
		},
		{
			"Keys passing keys but failing list fails",
			map[string][]string{
				"good":     {"this is bxd", "this is not gxxd"},
				"alsogood": {"doesn't matter, already failing"},
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			c := Config{
				schema: schema{
					Validate: test.validateList,
				},
				selector: mockEnvSelector{},
			}

			v := c.ValidateEnv()

			if v == nil && !test.shouldValidate {
				t.Error("Validate passed when it shouldn't have")
			} else if v != nil && test.shouldValidate {
				t.Errorf("Validate failed with %s", v)
			}
		})
	}
}

func TestCheckAndInstallCmd(t *testing.T) {
	tests := []struct {
		desc       string
		env        mockEnvSelector
		checkCmd   []string
		installCmd []string
	}{
		{
			"Environment matches distro-one",
			mockEnvSelector{"distro-one", []string{}},
			[]string{"pkgmgr", "check"},
			[]string{"pkgmgr", "install"},
		},
		{
			"Environment matches t..t",
			mockEnvSelector{"t..t", []string{}},
			[]string{"testpkg-check"},
			[]string{"testpkg-install"},
		},
		{
			"Environment does not match",
			mockEnvSelector{"", []string{}},
			nil,
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := Config{goodSchema, test.env}
			checkCmd := config.CheckCmd()
			if !reflect.DeepEqual(test.checkCmd, checkCmd) {
				t.Errorf(
					"expected checkCmd to be %#v, got %#v",
					test.checkCmd,
					checkCmd,
				)
			}

			installCmd := config.InstallCmd()
			if !reflect.DeepEqual(test.installCmd, installCmd) {
				t.Errorf(
					"expected installCmd to be %#v, got %#v",
					test.installCmd,
					installCmd,
				)
			}
		})
	}
}

func TestPackagesRealDotfile(t *testing.T) {
	tests := []struct {
		desc    string
		env     mockEnvSelector
		pkgList []Package
	}{
		{
			"Regexp empty",
			mockEnvSelector{"test(s?)", []string{"test"}},
			[]Package{
				{"foo", "For tests", []string{"bar", "baz"}},
			},
		},
		{
			`Regexp with "s"`,
			mockEnvSelector{"test(s?)", []string{"tests"}},
			[]Package{
				{"foo", "For tests", []string{"bars", "baz"}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := Config{goodSchema, test.env}
			packages, err := config.Packages("dotfile")
			if err != nil {
				t.Fatal("expected err to be nil, got non-nil")
			}
			if !reflect.DeepEqual(test.pkgList, packages) {
				t.Fatalf(
					"expected package list to be %#v, got %#v",
					test.pkgList,
					packages,
				)
			}

		})
	}
}

func TestPackagesBadDotfile(t *testing.T) {
	config := Config{goodSchema, mockEnvSelector{}}
	_, err := config.Packages("not-real-dotfile")
	if err == nil {
		t.Fatal("expected err to be non-nil, got nil")
	}
}

func TestDotConfigRealDotfile(t *testing.T) {
	tests := []struct {
		desc    string
		env     mockEnvSelector
		dotName string
		dotConf DotConfig
	}{
		{
			"test env with dotfile dot",
			mockEnvSelector{"test", []string{"test"}},
			"dotfile",
			DotConfig{
				Method:       "copy",
				Root:         "/test/dir/:",
				DotPrefix:    false,
				dotPrefixSet: true,
				Rules: map[string]string{
					"dir/file": "/new/loc/file",
				},
			},
		},
		{
			"No env with dotfile dot",
			mockEnvSelector{"bad-key", []string{}},
			"dotfile",
			DotConfig{
				Method:       "copy",
				Root:         "/test/dir/:",
				DotPrefix:    true,
				dotPrefixSet: false,
				Rules:        nil,
			},
		},
		{
			"test env with commonless dot",
			mockEnvSelector{"test", []string{"test"}},
			"commonless",
			DotConfig{
				Method:       "shallow",
				Root:         "home",
				DotPrefix:    false,
				dotPrefixSet: true,
				Rules:        nil,
			},
		},
		{
			"No env with commonless dot",
			mockEnvSelector{"bad-key", []string{}},
			"commonless",
			DotConfig{
				Method:       "deep",
				Root:         "xdg",
				DotPrefix:    true,
				dotPrefixSet: false,
				Rules:        nil,
			},
		},
		{
			"Dot with templated rules",
			mockEnvSelector{"template-(.*)", []string{"template-xyz"}},
			"templated",
			DotConfig{
				Method:       "deep",
				Root:         "xdg",
				DotPrefix:    true,
				dotPrefixSet: false,
				Rules: map[string]string{
					"file-xyz": "/outfile",
				},
			},
		},
		{
			"Nonexistant dot falls back to global and env config",
			mockEnvSelector{"test", []string{"test"}},
			"bad-dot",
			DotConfig{
				Method:       "shallow",
				Root:         "home",
				DotPrefix:    false,
				dotPrefixSet: true,
				Rules:        nil,
			},
		},
		{
			"Nonexistant dot falls back to global config",
			mockEnvSelector{"non-existant key", []string{}},
			"bad-dot",
			DotConfig{
				Method:       "deep",
				Root:         "xdg",
				DotPrefix:    true,
				dotPrefixSet: false,
				Rules:        nil,
			},
		},
		{
			"Deployable dot has deploy populated",
			mockEnvSelector{"test", []string{"test"}},
			"deployable",
			DotConfig{
				Method:       "shallow",
				Root:         "home",
				DotPrefix:    false,
				dotPrefixSet: true,
				Rules:        nil,
				Deploy: [][]string{
					{"command", "one"},
					{"cmd", "$TWO"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := Config{goodSchema, test.env}
			dotConf := config.DotConfig(test.dotName)

			if !reflect.DeepEqual(test.dotConf, dotConf) {
				t.Fatalf(
					"expected dotConf to be %#v, got %#v",
					test.dotConf,
					dotConf,
				)
			}
		})
	}
}
