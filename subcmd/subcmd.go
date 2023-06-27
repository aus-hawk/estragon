package subcmd

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/aus-hawk/estragon/config"
	"github.com/aus-hawk/estragon/logging"
	"github.com/aus-hawk/estragon/subcmd/install"
)

type SubcmdRunner struct {
	conf config.Config
	dry  bool
}

func NewSubcmdRunner(conf config.Config, dry bool) SubcmdRunner {
	return SubcmdRunner{conf, dry}
}

func (s SubcmdRunner) RunSubCmd(subcmd string, dots []string) error {
	switch subcmd {
	case "install":
		return s.installSubCmd(dots)
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
