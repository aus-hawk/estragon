package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/aus-hawk/estragon/config"
	"github.com/aus-hawk/estragon/env"
	"github.com/aus-hawk/estragon/subcmd"
)

func main() {
	args, err := parseFlags()
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			// Passing "-h" isn't actually an error.
			fmt.Fprintln(os.Stderr, "Error parsing flags:", err)
			os.Exit(1)
		}
		return
	}

	if args.dry && args.subcommand != "envvar" {
		fmt.Println("Running in dry mode, no changes will be made")
	}

	dir, err := initDir(args.dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing dir:", err)
		os.Exit(1)
	}

	env, err := getEnv(args.env, dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting environment:", err)
		os.Exit(1)
	}

	conf, err := getConfig(dir, env)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting config:", err)
		os.Exit(1)
	}

	if args.subcommand != "envvar" {
		fmt.Printf("Using environment: %s\n\n", env)
	}

	runner := subcmd.NewSubcmdRunner(conf, dir, args.dry, args.force)

	dots := removeDuplicates(args.dots)

	if args.all {
		dots = append(conf.AllDots(), dots...)
		dots = removeDuplicates(dots)
	}

	err = runner.RunSubcmd(args.subcommand, dots)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
}

type cmdArgs struct {
	subcommand, dir, env string
	dry, force, all      bool
	dots                 []string
}

func parseFlags() (args cmdArgs, err error) {
	subcmdFlags := flag.NewFlagSet("subcommands", flag.ContinueOnError)
	subcmdFlags.SetOutput(os.Stderr)
	subcmdFlags.Usage = func() {
		helpLines := []string{
			"usage: estragon [subcommand] [options] [dots|envs]",
			"",
			"subcommands:",
			"  install  - Install the packages of each dot",
			"  deploy   - Deploy the files in the dot folders",
			"  undeploy - Delete files that were previously deployed",
			"  redeploy - Undeploy, then deploy each dot",
			"  envvar   - Set and print local environment variables",
			"  help     - Display this message",
			"",
			"All subcommands take a list of dots except for envvar,",
			"which takes strings without equal signs to print an",
			"environment value, with equal signs to set them to new",
			"values, and with a minus (-) after the name to remove them",
			"",
			"A lack of a subcommand will print the ownership of",
			"the dots (all of them by default) and store the",
			"environment you pass",
		}
		for _, l := range helpLines {
			fmt.Println(l)
		}
		fmt.Println()
		fmt.Println("options:")
		subcmdFlags.PrintDefaults()
	}

	dir := subcmdFlags.StringP(
		"dir",
		"d",
		"",
		"The `directory` containing the dots (current one by default)",
	)

	env := subcmdFlags.StringP(
		"env",
		"e",
		"",
		"The `environment` string used in environment matching",
	)

	dry := subcmdFlags.BoolP(
		"dry",
		"n",
		false,
		"Show what the command would do without changing the system",
	)

	force := subcmdFlags.BoolP(
		"force",
		"f",
		false,
		"Force ownership of files on deploy, overwriting existing ones",
	)

	all := subcmdFlags.BoolP(
		"all",
		"a",
		false,
		"Add all dots defined in estragon.yaml to the dot list",
	)

	var argList []string

	if len(os.Args) < 2 {
		// Called with no subcommand or flags.
		return
	} else if strings.HasPrefix(os.Args[1], "-") {
		// Called with no subcommand.
		argList = os.Args[1:]
	} else if os.Args[1] == "help" {
		subcmdFlags.Usage()
		err = flag.ErrHelp
		return
	} else {
		args.subcommand = os.Args[1]
		argList = os.Args[2:]
	}

	err = subcmdFlags.Parse(argList)
	if err != nil {
		return
	}

	args.dir = *dir
	args.env = *env
	args.dry = *dry
	args.force = *force
	args.all = *all
	args.dots = subcmdFlags.Args()
	return
}

func initDir(argDir string) (dir string, err error) {
	dir = argDir
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	if dir == "" {
		dir = wd
	} else if filepath.IsLocal(dir) {
		dir = filepath.Join(wd, dir)
	}

	for dir != filepath.Dir(dir) {
		estragonYaml := filepath.Join(dir, "estragon.yaml")
		if _, err := os.Stat(estragonYaml); errors.Is(err, os.ErrNotExist) {
			dir = filepath.Dir(dir)
		} else if err != nil {
			return dir, err
		} else {
			// Found the file.
			break
		}
	}

	estragonYaml := filepath.Join(dir, "estragon.yaml")
	if _, err := os.Stat(estragonYaml); errors.Is(err, os.ErrNotExist) {
		err = errors.New("No estragon.yaml file in directory or parents")
		return dir, err
	} else if err != nil {
		return dir, err
	}

	estragonDir := filepath.Join(dir, ".estragon")

	err = os.Mkdir(estragonDir, 0777)
	if errors.Is(err, fs.ErrExist) {
		err = nil
	} else if err != nil {
		return
	}

	gitignore := filepath.Join(estragonDir, ".gitignore")
	err = os.WriteFile(gitignore, []byte("*"), 0666)
	if err != nil {
		return
	}

	ownJson := filepath.Join(estragonDir, "own.json")
	if _, err := os.Stat(ownJson); errors.Is(err, os.ErrNotExist) {
		// Ensure that the ownership file exists and has an empty
		// object.
		err = os.WriteFile(ownJson, []byte("{}"), 0666)
	} else if err != nil {
		return dir, err
	}

	envvars := filepath.Join(estragonDir, "envvars")
	f, err := os.OpenFile(envvars, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	return
}

func getEnv(argEnv, dir string) (env string, err error) {
	env = argEnv

	envFile := filepath.Join(dir, ".estragon", "env")

	if env == "" {
		var envBytes []byte
		envBytes, err = os.ReadFile(envFile)
		if err != nil {
			err = errors.New(
				"No --env argument or .estragon/env file in directory",
			)
			return
		}
		env = string(envBytes)
	} else {
		err = os.WriteFile(envFile, []byte(env), 0666)
	}

	return
}

func getConfig(dir, envString string) (conf config.Config, err error) {
	confFile := filepath.Join(dir, "estragon.yaml")

	f, err := os.ReadFile(confFile)
	if err != nil {
		return
	}

	environment := env.NewEnvironment(envString)
	conf, err = config.NewConfig(f, environment)

	return
}

func removeDuplicates(s []string) []string {
	unique := make(map[string]struct{})
	uniqueS := make([]string, 0)
	for _, x := range s {
		if _, ok := unique[x]; !ok {
			unique[x] = struct{}{}
			uniqueS = append(uniqueS, x)
		}
	}
	return uniqueS
}
