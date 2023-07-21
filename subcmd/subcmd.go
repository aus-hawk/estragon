package subcmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aus-hawk/estragon/config"
)

type SubcmdRunner struct {
	conf       config.Config
	dir        string
	dry, force bool
}

func NewSubcmdRunner(conf config.Config, dir string, dry, force bool) SubcmdRunner {
	return SubcmdRunner{conf, dir, dry, force}
}

func (s SubcmdRunner) RunSubcmd(subcmd string, dots []string) error {
	err := s.conf.ValidateEnv()
	if err != nil && subcmd != "envvar" {
		return err
	}

	if subcmd != "envvar" {
		envvars, err := s.getEnvvars()
		if err != nil {
			return err
		}
		for k, v := range envvars {
			err := os.Setenv(k, v)
			if err != nil {
				return err
			}
		}
	}

	switch subcmd {
	case "install":
		pkgInstaller, err := NewPackageInstaller(s.conf, s.dry)
		if err != nil {
			return err
		}
		return pkgInstaller.Install(dots)
	case "deploy":
		return s.deploySubcmd(dots)
	case "undeploy":
		return s.undeploySubcmd(dots)
	case "redeploy":
		fmt.Print("Undeploying dots\n\n")
		err := s.undeploySubcmd(dots)
		if err != nil {
			return err
		}
		fmt.Print("\n\n")
		fmt.Print("Deploying dots\n\n")
		return s.deploySubcmd(dots)
	case "envvar":
		envvars, err := s.getEnvvars()
		if err != nil {
			return err
		}
		return Envvars(dots, envvars, s.dir)
	case "":
		return s.printOwnership(dots)
	default:
		errMsg := fmt.Sprintf(`"%s" is not a valid subcommand`, subcmd)
		return errors.New(errMsg)
	}
}

func runCmd(args []string) (int, error) {
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()
	if err == nil {
		// Run was normal and successful.
		return 0, nil
	} else if exitError, ok := err.(*exec.ExitError); ok {
		// Ignore the error, only use it for the exit code.
		exitCode := exitError.ExitCode()
		return exitCode, nil
	} else {
		return -1, err
	}
}

func (s SubcmdRunner) deploySubcmd(dots []string) error {
	ownJson := filepath.Join(s.dir, ".estragon", "own.json")
	own := OwnershipManager{ownJson, s.force}
	for i, dot := range dots {
		conf := s.conf.DotConfig(dot)
		root := filepath.Join(s.dir, dot)
		deployer := NewDotfileDeployer(
			conf,
			root,
			pathExpander{dot}.expand,
			own,
			s.dry,
		)

		files, err := dirFiles(root)
		if err != nil {
			return err
		}

		err = deployer.DeployFiles(dot, files)
		if err != nil {
			return err
		}

		fmt.Println()

		err = deployer.DeployCmd(dot)
		if err != nil {
			return err
		}

		if i != len(dots)-1 {
			fmt.Println()
		}
	}

	return nil
}

func dirFiles(dotDir string) ([]string, error) {
	if _, err := os.Stat(dotDir); errors.Is(err, os.ErrNotExist) {
		// Don't try to walk or an error occurs. It's possible to want
		// to deploy a dot without there being a dot folder.
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	files := make([]string, 0)

	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			relPath, err := filepath.Rel(dotDir, path)
			if err != nil {
				return err
			}
			relPath = filepath.ToSlash(relPath)
			files = append(files, relPath)
		}
		return nil
	}

	err := filepath.WalkDir(dotDir, walkFunc)
	return files, err
}

type pathExpander struct {
	dot string
}

func (p pathExpander) expand(s string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return s, err
	}

	s = strings.ReplaceAll(s, "~", home)
	s = strings.ReplaceAll(s, "*", p.dot)
	s = os.Expand(s, func(k string) string {
		e, varExists := os.LookupEnv(k)
		if err == nil && !varExists {
			err = errors.New(
				"Environment variable " + k + " is not set",
			)
		}
		return e
	})
	return s, err
}

func (s SubcmdRunner) undeploySubcmd(dots []string) error {
	ownJson := filepath.Join(s.dir, ".estragon", "own.json")
	own := OwnershipManager{ownJson, false}
	undeployer := DotfileUndeployer{own, s.dry}
	for i, dot := range dots {
		err := undeployer.Undeploy(dot)
		if err != nil {
			return err
		}
		if i != len(dots)-1 {
			fmt.Println()
		}
	}
	return nil
}

func (s SubcmdRunner) getEnvvars() (map[string]string, error) {
	varMap := make(map[string]string)

	envvars := filepath.Join(s.dir, ".estragon", "envvars")
	varsBytes, err := os.ReadFile(envvars)
	if err != nil {
		return varMap, err
	}

	vars := strings.Split(string(varsBytes), "\n")
	for _, v := range vars {
		if v == "" {
			continue
		}
		pair := strings.SplitN(v, "=", 2)
		if len(pair) == 2 {
			varMap[pair[0]] = pair[1]
		} else {
			return varMap, errors.New(
				".estragon/envvars file is incorrectly formatted",
			)
		}
	}

	return varMap, nil
}

func (s SubcmdRunner) printOwnership(dots []string) error {
	ownJson := filepath.Join(s.dir, ".estragon", "own.json")
	own := OwnershipManager{ownJson, false}

	dotOwn, err := own.OwnedFiles()
	if err != nil {
		return err
	}

	if len(dots) > 0 {
		allOwnedDots := dotOwn
		dotOwn = make(map[string][]string)
		for _, dot := range dots {
			dotOwn[dot] = allOwnedDots[dot]
		}
	}

	for dot, owned := range dotOwn {
		fmt.Printf("Files owned by %s:\n", dot)
		for _, o := range owned {
			fmt.Printf("  %s\n", o)
		}
	}

	return nil
}
