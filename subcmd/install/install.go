package install

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

type CmdRunner func(cmd []string) (int, error)

type PackageInstaller struct {
	mgr PackageManager
	run CmdRunner
}

// NewPackageInstaller creates a package installer that manages packages with
// mgr and runs the commands to check for and install them with the run
// function.
func NewPackageInstaller(mgr PackageManager, run CmdRunner) PackageInstaller {
	return PackageInstaller{mgr, run}
}

// Install will attempt to install the packages that belong to a list of dots.
// Those dots will be used to query the package manager that the installer was
// created with. It will print out the installation progress to standard out. If
// something goes wrong while querying the package manager or running commands,
// an error will be returned, otherwise it will be nil. If dry is true, no
// installation actually occurs. Instead, the commands used and the packages
// that will be installed are printed to standard out.
func (p PackageInstaller) Install(dots []string, dry bool) error {
	checkCmd := strings.Join(p.mgr.CheckCmd(), " ")
	fmt.Printf("Matching check-cmd: %s [PACKAGE]\n", checkCmd)

	installCmd := strings.Join(p.mgr.InstallCmd(), " ")
	fmt.Printf("Matching install-cmd: %s [PACKAGE]\n", installCmd)

	if dry {
		p.dryInstall(dots)
		return nil
	}

	for _, dot := range dots {
		fmt.Printf("Installing packages for %s\n", dot)
		err := p.installDot(dot)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p PackageInstaller) installDot(dot string) error {
	pkgs, err := p.mgr.Packages(dot)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		fmt.Printf("  %s: %s\n", pkg.Name, pkg.Desc)
		for _, realPkg := range pkg.List {
			err := p.installPackage(realPkg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p PackageInstaller) installPackage(pkg string) error {
	fmt.Printf("    Installing %s...", pkg)
	checkCmd := append(p.mgr.CheckCmd(), pkg)
	code, err := p.run(checkCmd)
	if err != nil {
		fmt.Println()
		return err
	} else if code == 0 {
		// Check successful, the package is already installed.
		fmt.Println("already installed")
		return nil
	}

	// At this point, the package is known to not be installed.
	installCmd := append(p.mgr.InstallCmd(), pkg)
	code, err = p.run(installCmd)
	if err != nil {
		fmt.Println()
		return err
	} else if code != 0 {
		fmt.Println()
		cmd := strings.Join(installCmd, " ")
		errMsg := fmt.Sprintf("%s exited with %d", cmd, code)
		return errors.New(errMsg)
	} else {
		fmt.Println("installed")
	}

	return nil
}

func (p PackageInstaller) dryInstall(dots []string) {
	for _, dot := range dots {
		fmt.Printf("Packages for dot %s and their expansions:", dot)
		pkgs, err := p.mgr.Packages(dot)
		if err != nil {
			fmt.Println(err)
		}
		for _, pkg := range pkgs {
			pkgList := strings.Join(pkg.List, " ")
			fmt.Printf("  %s -> %s", pkg.Name, pkgList)
		}
	}
}
