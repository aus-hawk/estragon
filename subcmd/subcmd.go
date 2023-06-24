package subcmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/aus-hawk/estragon/config"
	"github.com/aus-hawk/estragon/subcmd/install"
)

type SubcmdRunner struct {
	conf        config.Config
	dry         bool
	infoLogger  *log.Logger
	errorLogger *log.Logger
}

func NewSubcmdRunner(
	conf config.Config,
	dry bool,
	logFile io.Writer,
) *SubcmdRunner {
	return &SubcmdRunner{
		conf,
		dry,
		log.New(logFile, "[INFO] ", log.LstdFlags),
		log.New(logFile, "[ERROR] ", log.LstdFlags),
	}
}

func (s *SubcmdRunner) RunSubCmd(subcmd string, dots []string) error {
	switch subcmd {
	case "install":
		return s.installSubCmd(dots)
	default:
		errMsg := fmt.Sprintf(`"%s" is not a valid subcommand`, subcmd)
		return errors.New(errMsg)
	}
}

func (s *SubcmdRunner) runCmd(args []string) (int, error) {
	cmd := exec.Command(args[0], args[1:]...)
	s.infoLogger.Println(strings.Join(args, " "))
	err := cmd.Run()
	if err != nil {
		return 0, nil
	}

	s.errorLogger.Println(err)
	if exitError, ok := err.(*exec.ExitError); ok {
		// Ignore the error, only use it for the exit code.
		return exitError.ExitCode(), nil
	} else {
		return -1, err
	}
}

func (s *SubcmdRunner) installSubCmd(dots []string) error {
	checkCmd := s.conf.CheckCmd()
	if len(checkCmd) == 0 {
		return errors.New("No matching non-empty check-cmd for environment")
	}

	installCmd := s.conf.InstallCmd()
	if len(installCmd) == 0 {
		return errors.New("No matching non-empty install-cmd for environment")
	}

	pkgInstaller := install.NewPackageInstaller(s.conf, s.runCmd)

	return pkgInstaller.Install(dots, s.dry)
}
