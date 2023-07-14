package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

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

	fmt.Printf("Using environment: %s\n\n", env)

	runner := subcmd.NewSubcmdRunner(conf, dir, args.dry, args.force)

	err = runner.RunSubcmd(args.subcommand, args.dots)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
}

type cmdArgs struct {
	subcommand, dir, env string
	dry, force           bool
	dots                 []string
}

func parseFlags() (args cmdArgs, err error) {
	subcmdFlags := flag.NewFlagSet("subcommands", flag.ContinueOnError)
	subcmdFlags.SetOutput(os.Stderr)
	subcmdFlags.Usage = func() {
		fmt.Println(
			"usage: estragon [subcommand] [options] [dots]",
		)
		fmt.Println()
		fmt.Println("subcommands:")
		fmt.Println("  install  - Install the packages of each dot")
		fmt.Println("  deploy   - Deploy the files in the dot folders")
		fmt.Println("  undeploy - Delete files that were previously deployed")
		fmt.Println("  redeploy - Undeploy, then deploy each dot")
		fmt.Println("  help     - Display this message")
		fmt.Println()
		fmt.Println("options:")
		subcmdFlags.PrintDefaults()
	}

	dir := subcmdFlags.String(
		"dir",
		"",
		"The `directory` containing the dots (current one by default)",
	)

	env := subcmdFlags.String(
		"env",
		"",
		"The `environment` string used in environment matching",
	)

	dry := subcmdFlags.Bool(
		"dry",
		false,
		"Show what the command would do without changing the system",
	)

	force := subcmdFlags.Bool(
		"force",
		false,
		"Force ownership of files on deploy, overwriting existing ones",
	)

	if len(os.Args) < 2 {
		subcmdFlags.Usage()
		err = errors.New("Subcommand not specified")
		return
	} else if !subcmd.ValidSubcmd(os.Args[1]) {
		subcmdFlags.Usage()
		match, _ := regexp.Match("^--?h(elp)?$", []byte(os.Args[1]))
		if match || os.Args[1] == "help" {
			err = flag.ErrHelp
		}
		return
	}

	args.subcommand = os.Args[1]
	err = subcmdFlags.Parse(os.Args[2:])
	if err != nil {
		return
	}

	args.dir = *dir
	args.env = *env
	args.dry = *dry
	args.force = *force
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
				"No -env passed or .estragon/env file in directory",
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
