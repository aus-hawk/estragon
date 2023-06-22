package subcmd

import (
	"errors"
	"reflect"
	"testing"

	"github.com/aus-hawk/estragon/config"
)

type mockPackageManager struct {
	check   []string
	install []string
	pkgs    []config.Package
	err     error
}

func (m mockPackageManager) CheckCmd() []string {
	return m.check
}

func (m mockPackageManager) InstallCmd() []string {
	return m.install
}

func (m mockPackageManager) Packages(_ string) ([]config.Package, error) {
	return m.pkgs, m.err
}

type mockCmdRet struct {
	code     int
	err      error
	lastArgs []string
}

type mockCmdRunner struct {
	checkRet   mockCmdRet
	installRet mockCmdRet
	t          *testing.T
}

func (m mockCmdRunner) Run(cmd []string) (int, error) {
	lastArg := cmd[len(cmd)-1]
	if len(cmd) >= 2 && cmd[1] == "install" {
		if !inSlice(lastArg, m.installRet.lastArgs) {
			m.t.Fatalf(
				"last arg to install was %#v, expected one of %#v",
				lastArg,
				m.installRet.lastArgs,
			)
		}
		return m.installRet.code, m.installRet.err
	} else {
		if !inSlice(lastArg, m.checkRet.lastArgs) {
			m.t.Fatalf(
				"last arg to check was %#v, expected one of %#v",
				lastArg,
				m.checkRet.lastArgs,
			)
		}
		return m.checkRet.code, m.checkRet.err
	}
}

func inSlice(s string, ss []string) bool {
	for _, x := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func TestNewPackageInstaller(t *testing.T) {
	expected := PackageInstaller{mockPackageManager{}, mockCmdRunner{}}
	actual := NewPackageInstaller(mockPackageManager{}, mockCmdRunner{})
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %#v, got %#v", expected, actual)
	}
}

func TestInstall(t *testing.T) {
	tests := []struct {
		desc   string
		mgr    mockPackageManager
		runner mockCmdRunner
		err    string
	}{
		{
			"Invalid check cmd",
			mockPackageManager{},
			mockCmdRunner{},
			"No matching check-cmd for environment",
		},
		{
			"Invalid install cmd",
			mockPackageManager{
				check: []string{"pkg", "check"},
			},
			mockCmdRunner{},
			"No matching install-cmd for environment",
		},
		{
			"Error getting packages",
			mockPackageManager{
				check:   []string{"pkg", "check"},
				install: []string{"pkg", "install"},
				err:     errors.New("bad dot"),
			},
			mockCmdRunner{},
			"bad dot",
		},
		{
			"Bad check",
			mockPackageManager{
				check:   []string{"pkg", "check"},
				install: []string{"pkg", "install"},
				pkgs: []config.Package{
					{
						Name: "abc",
						Desc: "123",
						List: []string{"do", "re", "me"},
					},
				},
			},
			mockCmdRunner{
				checkRet: mockCmdRet{
					code:     0,
					err:      errors.New("bad check"),
					lastArgs: []string{"do"},
				},
			},
			"bad check",
		},
		{
			"Already installed",
			mockPackageManager{
				check:   []string{"pkg", "check"},
				install: []string{"pkg", "install"},
				pkgs: []config.Package{
					{
						Name: "abc",
						Desc: "123",
						List: []string{"do", "re", "me"},
					},
				},
			},
			mockCmdRunner{
				checkRet: mockCmdRet{
					code:     0,
					err:      nil,
					lastArgs: []string{"do", "re", "me"},
				},
			},
			"",
		},
		{
			"Bad install",
			mockPackageManager{
				check:   []string{"pkg", "check"},
				install: []string{"pkg", "install"},
				pkgs: []config.Package{
					{
						Name: "abc",
						Desc: "123",
						List: []string{"do", "re", "me"},
					},
				},
			},
			mockCmdRunner{
				// should fail at first package, which is do
				checkRet: mockCmdRet{
					code:     1,
					err:      nil,
					lastArgs: []string{"do"},
				},
				installRet: mockCmdRet{
					code:     0,
					err:      errors.New("bad install"),
					lastArgs: []string{"do"},
				},
			},
			"bad install",
		},
		{
			"Good install",
			mockPackageManager{
				check:   []string{"pkg", "check"},
				install: []string{"pkg", "install"},
				pkgs: []config.Package{
					{
						Name: "abc",
						Desc: "123",
						List: []string{"do", "re", "me"},
					},
				},
			},
			mockCmdRunner{
				checkRet: mockCmdRet{
					code:     1,
					err:      nil,
					lastArgs: []string{"do", "re", "me"},
				},
				installRet: mockCmdRet{
					code:     0,
					err:      nil,
					lastArgs: []string{"do", "re", "me"},
				},
			},
			"",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test.runner.t = t

			pkgInst := PackageInstaller{test.mgr, test.runner}
			// arg doesn't matter for testing
			err := pkgInst.Install("")

			if err != nil {
				if err.Error() != test.err {
					t.Fatalf(
						"expected err.Error() to be %#v, was %#v",
						test.err,
						err.Error(),
					)
				}
				return
			} else if test.err != "" {
				t.Fatal("expected err to be non-nil, was nil")
			}
		})
	}
}
