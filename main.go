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
	flags, err := parseFlags()
	if err != nil {
		if err != flag.ErrHelp {
			// Passing "-h" isn't actually an error.
			fmt.Fprintln(os.Stderr, "Error parsing flags:", err)
			os.Exit(1)
		}
		return
	}

	commandConf, err := getCommandConfig(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing flags:", err)
		os.Exit(1)
	}

	fmt.Printf("Using environment: %s\n\n", commandConf.env)

	conf, err := getConfig(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting config:", err)
		os.Exit(1)
	}

	runner := subcmd.NewSubcmdRunner(conf, commandConf.dir, flags.dry)

	err = runner.RunSubCmd(flags.subcommand, flags.dots)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
}

type cmdFlags struct {
	subcommand, dir, env string
	dry                  bool
	dots                 []string
}

func parseFlags() (f cmdFlags, err error) {
	subcmdFlags := flag.NewFlagSet("subcommands", flag.ContinueOnError)
	subcmdFlags.SetOutput(os.Stderr)
	subcmdFlags.Usage = func() {
		fmt.Println(
			"usage: estragon [install|deploy|help] [options] [dots]",
		)
		fmt.Println()
		fmt.Println("install - Install the packages of each dot")
		fmt.Println("deploy  - Deploy the files in the dot folders")
		fmt.Println("help    - Display this message")
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

	f.subcommand = os.Args[1]
	err = subcmdFlags.Parse(os.Args[2:])
	if err != nil {
		return
	}

	f.dir = *dir
	f.env = *env
	f.dry = *dry
	f.dots = subcmdFlags.Args()
	return
}

type cmdConf struct {
	dir, env string
}

func getCommandConfig(flags cmdFlags) (c cmdConf, err error) {
	c.dir = flags.dir
	if c.dir == "" {
		c.dir, err = os.Getwd()
		if err != nil {
			return
		}
	}

	c.env = flags.env
	estragonDir := filepath.Join(c.dir, ".estragon")
	envFile := filepath.Join(estragonDir, "env")
	if c.env == "" {
		env, err := os.ReadFile(envFile)
		if err != nil {
			return c, errors.New(
				"No -env passed or .estragon/env file in directory",
			)
		}
		c.env = string(env)
	} else {
		// Write most recently used env to storage.
		err = os.Mkdir(estragonDir, 0777)
		if err != nil && !errors.Is(err, fs.ErrExist) {
			return
		}

		err = os.WriteFile(envFile, []byte(c.env), 0666)
	}

	return
}

func getConfig(flags cmdFlags) (conf config.Config, err error) {
	confFile := filepath.Join(flags.dir, "estragon.yaml")

	f, err := os.ReadFile(confFile)
	if err != nil {
		return
	}

	env := env.NewEnvironment(flags.env)
	conf, err = config.NewConfig(f, env)

	return
}
