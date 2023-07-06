package subcmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aus-hawk/estragon/config"
	"github.com/aus-hawk/estragon/logging"
	"github.com/aus-hawk/estragon/subcmd/deploy"
	"github.com/aus-hawk/estragon/subcmd/install"
)

type SubcmdRunner struct {
	conf config.Config
	dir  string
	dry  bool
}

func NewSubcmdRunner(conf config.Config, dir string, dry bool) SubcmdRunner {
	return SubcmdRunner{conf, dir, dry}
}

func (s SubcmdRunner) RunSubCmd(subcmd string, dots []string) error {
	switch subcmd {
	case "install":
		return s.installSubCmd(dots)
	case "deploy":
		return s.deploySubCmd(dots)
	default:
		errMsg := fmt.Sprintf(`"%s" is not a valid subcommand`, subcmd)
		return errors.New(errMsg)
	}
}

func (s SubcmdRunner) installSubCmd(dots []string) error {
	checkCmd := s.conf.CheckCmd()
	if len(checkCmd) == 0 {
		return errors.New("No matching non-empty check-cmd for environment")
	}

	installCmd := s.conf.InstallCmd()
	if len(installCmd) == 0 {
		return errors.New("No matching non-empty install-cmd for environment")
	}

	pkgInstaller := install.NewPackageInstaller(s.conf, runCmd)

	return pkgInstaller.Install(dots, s.dry)
}

func runCmd(args []string) (int, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmdName := strings.Join(args, " ")
	logging.InfoLogger.Println(cmdName)
	err := cmd.Run()
	if err != nil {
		return 0, nil
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		// Ignore the error, only use it for the exit code.
		exitCode := exitError.ExitCode()
		logging.InfoLogger.Printf("%s returned %d", cmdName, exitCode)
		return exitCode, nil
	} else {
		logging.ErrorLogger.Println(err)
		return -1, err
	}
}

type fileDeployer struct{}

func (_ fileDeployer) Copy(m map[string]string) error {
	for k, v := range m {
		logging.InfoLogger.Printf("Copying file %s to %s\n", k, v)
		src, err := os.Open(k)
		if err != nil {
			logging.ErrorLogger.Println(err)
			return err
		}
		defer src.Close()

		if _, err := os.Stat(v); errors.Is(err, os.ErrNotExist) {
			dest, err := os.Create(v)
			if err != nil {
				logging.ErrorLogger.Println(err)
				return err
			}
			defer dest.Close()

			_, err = io.Copy(src, dest)
			if err != nil {
				logging.ErrorLogger.Println(err)
				return err
			}

			err = dest.Sync()
			if err != nil {
				logging.ErrorLogger.Println(err)
				return err
			}

			logging.InfoLogger.Println("File copied")
		} else {
			logging.ErrorLogger.Println(v + " already exists")
			return errors.New(v + " already exists")
		}
	}

	return nil
}

func (_ fileDeployer) Symlink(m map[string]string) error {
	for k, v := range m {
		logging.InfoLogger.Printf("Creating symlink %s to %s", v, k)
		err := os.Symlink(k, v)
		if err != nil {
			logging.ErrorLogger.Println(err)
		}
	}
	return nil
}

func (_ fileDeployer) Expand(s string) string {
	s = strings.ReplaceAll(s, "~", "$HOME")
	return os.ExpandEnv(s)
}

func (s SubcmdRunner) deploySubCmd(dots []string) error {
	deployer := deploy.NewDotfileDeployer(s.conf, fileDeployer{}, s.dir)

	for _, dot := range dots {
		files, err := s.dirFiles(dot)
		if err != nil {
			return err
		}

		err = deployer.Deploy(dot, files, s.dry)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s SubcmdRunner) dirFiles(dot string) ([]string, error) {
	files := make([]string, 0)

	dotDir := filepath.Join(s.dir, dot)

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
