package subcmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aus-hawk/estragon/config"
)

type PackageManager interface {
	CheckCmd() []string
	InstallCmd() []string
	Packages(string) ([]config.Package, error)
}

type CmdRunner interface {
	Run(cmd []string) (int, error)
}

type PackageInstaller struct {
	mgr    PackageManager
	runner CmdRunner
}

// NewPackageInstaller creates a package installer that manages packages with
// mgr and runs the commands to check for and install them with runner.
func NewPackageInstaller(mgr PackageManager, runner CmdRunner) PackageInstaller {
	return PackageInstaller{mgr, runner}
}

// Install will attempt to install the packages that belong to a dot. That dot
// will be used to query the package manager that it was created with. It will
// print out the installation progress to standard out. If something goes wrong
// while querying the package manager or running commands, an error will be
// returned, otherwise it will be nil.
func (p PackageInstaller) Install(dot string) error {
	checkCmd := p.mgr.CheckCmd()
	if len(checkCmd) == 0 {
		return errors.New("No matching check-cmd for environment")
	}
	installCmd := p.mgr.InstallCmd()
	if len(installCmd) == 0 {
		return errors.New("No matching install-cmd for environment")
	}

	pkgs, err := p.mgr.Packages(dot)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		err := p.installPackage(pkg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p PackageInstaller) installPackage(pkg config.Package) error {
	fmt.Printf("%s: %s\n", pkg.Name, pkg.Desc)

	for _, realPkg := range pkg.List {
		fmt.Printf("  Installing %s...", realPkg)
		checkCmd := append(p.mgr.CheckCmd(), realPkg)
		code, err := p.runner.Run(checkCmd)
		if err != nil {
			fmt.Println()
			return err
		} else if code == 0 {
			// Check successful, the package is already installed.
			fmt.Println("already installed")
			return nil
		}

		// At this point, the package is known to not be installed.
		installCmd := append(p.mgr.InstallCmd(), realPkg)
		code, err = p.runner.Run(installCmd)
		if err != nil {
			fmt.Println()
			return err
		} else if code != 0 {
			fmt.Println()
			cmd := strings.Join(installCmd, " ")
			errMsg := fmt.Sprintf("%s returned %d", cmd, code)
			return errors.New(errMsg)
		} else {
			fmt.Println("installed")
		}
	}

	return nil
}
