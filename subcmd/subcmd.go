package subcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aus-hawk/estragon/config"
	"github.com/aus-hawk/estragon/subcmd/deploy"
	"github.com/aus-hawk/estragon/subcmd/install"
)

type SubcmdRunner struct {
	conf       config.Config
	dir        string
	dry, force bool
}

func NewSubcmdRunner(conf config.Config, dir string, dry, force bool) SubcmdRunner {
	return SubcmdRunner{conf, dir, dry, force}
}

func ValidSubcmd(subcmd string) bool {
	switch subcmd {
	case "install", "deploy", "undeploy", "redeploy":
		return true
	default:
		return false
	}
}

func (s SubcmdRunner) RunSubcmd(subcmd string, dots []string) error {
	switch subcmd {
	case "install":
		return s.installSubcmd(dots)
	case "deploy":
		return s.deploySubcmd(dots)
	case "undeploy":
		return s.undeploySubcmd(dots)
	case "redeploy":
		err := s.undeploySubcmd(dots)
		if err != nil {
			return err
		}
		fmt.Println()
		return s.deploySubcmd(dots)
	default:
		errMsg := fmt.Sprintf(`"%s" is not a valid subcommand`, subcmd)
		return errors.New(errMsg)
	}
}

func (s SubcmdRunner) installSubcmd(dots []string) error {
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
	own := filepath.Join(s.dir, ".estragon", "own.json")
	deployer := deploy.NewDotfileDeployer(
		s.conf,
		fileDeployer{own, s.force},
		s.dir,
		runCmd,
	)

	for _, dot := range dots {
		files, err := s.dirFiles(dot)
		if err != nil {
			return err
		}

		err = deployer.Deploy(dot, files, s.dry)
		if err != nil {
			return err
		}

		fmt.Println()
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

type fileDeployer struct {
	ownJson string
	force   bool
}

func (d fileDeployer) Copy(m map[string]string, dot string) error {
	err := d.ensureOwnership(m, dot)
	if err != nil {
		return err
	}

	for k, v := range m {
		src, err := os.Open(k)
		if err != nil {
			return err
		}
		defer src.Close()

		err = os.MkdirAll(filepath.Dir(v), 0777)
		if err != nil {
			return err
		}

		dest, err := os.Create(v)
		if err != nil {
			return err
		}
		defer dest.Close()

		_, err = io.Copy(dest, src)
		if err != nil {
			return err
		}

		err = dest.Sync()
		if err != nil {
			return err
		}
	}

	return nil
}

func (d fileDeployer) Symlink(m map[string]string, dot string) error {
	err := d.ensureOwnership(m, dot)
	if err != nil {
		return err
	}

	for k, v := range m {
		err := os.MkdirAll(filepath.Dir(v), 0777)
		if err != nil {
			return err
		}
		err = os.Symlink(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (_ fileDeployer) Expand(s string, dot string) string {
	home, _ := os.UserHomeDir()
	s = strings.ReplaceAll(s, "~", home)
	s = strings.ReplaceAll(s, "*", dot)
	return os.ExpandEnv(s)
}

func (d fileDeployer) ensureOwnership(m map[string]string, dot string) error {
	data, err := os.ReadFile(d.ownJson)
	if err != nil {
		return err
	}

	var dotOwn map[string][]string
	err = json.Unmarshal(data, &dotOwn)
	if err != nil {
		return errors.New(
			"Error unmarshalling own.json file: " + err.Error(),
		)
	}

	ownedFileList := dotOwn[dot]
	ownedFiles := make(map[string]struct{})
	for _, file := range ownedFileList {
		ownedFiles[file] = struct{}{}
	}

	outFiles := make([]string, 0, len(m))
	for _, v := range m {
		outFiles = append(outFiles, v)
	}

	for _, file := range outFiles {
		_, ok := ownedFiles[file]
		if !ok {
			// File is not owned.
			if _, err := os.Stat(file); err == nil {
				if d.force {
					// Delete the file/folder before any
					// attempts to link/copy.
					err := os.RemoveAll(file)
					if err != nil {
						return err
					}
				} else {
					return errors.New(
						file + " exists but is not owned by this directory",
					)
				}
			} else if errors.Is(err, os.ErrNotExist) {
				// File does not exist and is not owned. Take
				// ownership.
				ownedFileList = append(ownedFileList, file)
			} else {
				return err
			}
		}
	}

	dotOwn[dot] = ownedFileList

	data, err = json.Marshal(dotOwn)
	if err != nil {
		return err
	}

	err = os.WriteFile(d.ownJson, data, 0666)

	return err
}

func (s SubcmdRunner) undeploySubcmd(dots []string) error {
	ownJson := filepath.Join(s.dir, ".estragon", "own.json")

	data, err := os.ReadFile(ownJson)
	if err != nil {
		return err
	}

	var dotOwn map[string][]string
	err = json.Unmarshal(data, &dotOwn)
	if err != nil {
		return err
	}

	for _, dot := range dots {
		owned, ok := dotOwn[dot]
		if ok {
			for _, file := range owned {
				fmt.Println("Removing file " + file)
				if !s.dry {
					err = os.Remove(file)
					if err != nil {
						return err
					}

					err = removeEmptyParents(file)
					if err != nil {
						return err
					}
				}
			}
			delete(dotOwn, dot)
		}
	}

	if s.dry {
		fmt.Println()
		fmt.Println("Directories that would be empty after these removals")
		fmt.Println("as well as their parents will also be removed")
	}

	data, err = json.Marshal(dotOwn)
	if err != nil {
		return err
	}

	if !s.dry {
		err = os.WriteFile(ownJson, data, 0666)
	}

	return err
}

func removeEmptyParents(file string) error {
	dir := filepath.Dir(file)
	for dir != "." {
		dirFile, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer dirFile.Close()

		_, err = dirFile.Readdirnames(1)
		if errors.Is(err, io.EOF) {
			// Folder is empty.
			fmt.Println("Removing empty directory " + dir)
			err = os.Remove(dir)
			if err != nil {
				return err
			}
		} else {
			return err
		}

		dir = filepath.Dir(dir)
	}
	return nil
}
