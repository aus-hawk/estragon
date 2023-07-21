package subcmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aus-hawk/estragon/config"
)

type PackageInstaller struct {
	conf config.Config
	dry  bool
}

func NewPackageInstaller(
	conf config.Config,
	dry bool,
) (p PackageInstaller, err error) {
	if len(conf.CheckCmd()) == 0 {
		err = errors.New("Empty check command")
	} else if len(conf.InstallCmd()) == 0 {
		err = errors.New("Empty install command")
	}
	p = PackageInstaller{
		conf,
		dry,
	}
	return
}

func (p PackageInstaller) Install(dots []string) error {
	fmt.Printf(
		"Matching check-cmd: %s [PACKAGE]\n",
		strings.Join(p.conf.CheckCmd(), " "),
	)
	fmt.Printf(
		"Matching install-cmd: %s [PACKAGE]\n",
		strings.Join(p.conf.InstallCmd(), " "),
	)

	for i, dot := range dots {
		err := p.installDotPackages(dot)
		if err != nil {
			return err
		}
		if i != len(dots) {
			// Separate dots in output.
			fmt.Println()
		}
	}

	return nil
}

func (p PackageInstaller) installDotPackages(dot string) error {
	pkgs, err := p.conf.Packages(dot)
	if err != nil {
		return err
	}

	fmt.Println("Installing packages for", dot)
	for _, pkg := range pkgs {
		fmt.Printf("  %s: %s\n", pkg.Name, pkg.Desc)
		for _, realPkg := range pkg.List {
			if !p.dry {
				fmt.Printf("    Installing %s", realPkg)
				fmt.Print("...")
				newlyInstalled, err := p.installPkg(realPkg)
				if err != nil {
					fmt.Println()
					return err
				}

				if newlyInstalled {
					fmt.Println("installed")
				} else {
					fmt.Println("already installed")
				}
			} else {
				fmt.Printf("    Installing %s\n", realPkg)
			}
		}
	}

	return nil
}

// If the returned bool is true then the package was installed to the system by
// the function. If it is false, the package was already installed. If something
// goes wrong either way, the error is non-nil.
func (p PackageInstaller) installPkg(pkg string) (bool, error) {
	checkCmd := append(p.conf.CheckCmd(), pkg)
	code, err := runCmd(checkCmd)
	if code == 0 {
		// The package already is installed.
		return false, nil
	} else if err != nil {
		return false, err
	}

	installCmd := append(p.conf.InstallCmd(), pkg)
	code, err = runCmd(installCmd)
	if code == 0 {
		// The package installed successfully.
		return true, nil
	} else if err == nil {
		return true, fmt.Errorf(
			"Command %s exited with %d",
			strings.Join(installCmd, " "),
			code,
		)
	} else {
		return true, err
	}
}
