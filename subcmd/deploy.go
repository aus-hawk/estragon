package subcmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aus-hawk/estragon/config"
	"github.com/aus-hawk/estragon/dotfile"
)

type DotfileDeployer struct {
	conf     config.DotConfig
	resolver dotfile.Resolver
	expand   dotfile.PathExpander
	own      OwnershipManager
	dry      bool
}

// NewDotfileDeployer creates a DotfileDeployer that uses the conf to know how
// to resolve and deploy the files.
func NewDotfileDeployer(
	conf config.DotConfig,
	dotRoot string,
	expand dotfile.PathExpander,
	own OwnershipManager,
	dry bool,
) DotfileDeployer {
	resolver := dotfile.NewResolver(dotRoot, conf.Root, conf.DotPrefix, expand)
	return DotfileDeployer{conf, resolver, expand, own, dry}
}

// DeployFiles either copies or creates links of files within the dot file tree
// outside of that file tree. dot is the name of the dot that is being deployed.
// files is a slice of all of the files (not including directories) within the
// dot directory. An empty slice represents an empty directory, and a nil one
// represents a non-existent directory.
func (d DotfileDeployer) DeployFiles(dot string, files []string) error {
	method := d.conf.Method
	rules := d.conf.Rules
	root := d.conf.Root

	fmt.Println("Method:", method)
	if method != "none" {
		expandedRoot, err := d.expand(root)
		if err != nil {
			return err
		}

		if expandedRoot != root {
			expandedRoot += " (expanded from " + root + ")"
		}
		fmt.Println("Root:", expandedRoot)
		fmt.Println("Dot prefix:", d.conf.DotPrefix)
		if len(rules) > 0 {
			fmt.Println("Rules:")
			for k, v := range rules {
				fmt.Printf("  %s -> %s\n", k, v)
			}
		}
	}

	fileMap, err := d.resolve(files, rules)
	if err != nil {
		return err
	}

	fmt.Println()

	if len(fileMap) == 0 {
		// No deployable files is not an error.
		fmt.Println("No files to deploy")
		return nil
	}

	outFiles := make([]string, 0, len(fileMap))
	for _, outFile := range fileMap {
		outFiles = append(outFiles, outFile)
	}
	err = d.own.EnsureOwnership(outFiles, dot)
	if err != nil {
		return err
	}

	return d.deploy(fileMap)
}

func (d DotfileDeployer) resolve(
	files []string,
	rules map[string]string,
) (map[string]string, error) {
	switch d.conf.Method {
	case "deep", "copy":
		return d.resolver.DeepResolve(files, rules)
	case "shallow":
		if files != nil {
			// nil files means the directory doesn't exist and we
			// shouldn't link to a non-existent directory.
			return d.resolver.ShallowResolve(files, rules)
		} else {
			return nil, nil
		}
	case "none":
		return nil, nil
	default:
		return nil, errors.New(d.conf.Method + " is not a valid method")
	}
}

func (d DotfileDeployer) deploy(fileMap map[string]string) error {
	switch d.conf.Method {
	case "deep", "shallow":
		fmt.Println("Creating the following symlinks (link -> original):")
		for dotfile, link := range fileMap {
			fmt.Printf("  %s -> %s\n", link, dotfile)
			if !d.dry {
				err := symlink(dotfile, link)
				if err != nil {
					return err
				}
			}
		}
	case "copy":
		fmt.Println("Copying the following files (original -> copy):")
		for dotfile, copyFile := range fileMap {
			fmt.Printf("  %s -> %s\n", dotfile, copyFile)
			if !d.dry {
				err := copyDotfile(dotfile, copyFile)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func symlink(file, link string) error {
	err := os.MkdirAll(filepath.Dir(link), 0777)
	if err != nil {
		return err
	}
	return os.Symlink(file, link)
}

func copyDotfile(existing, newFile string) error {
	src, err := os.Open(existing)
	if err != nil {
		return err
	}
	defer src.Close()

	err = os.MkdirAll(filepath.Dir(newFile), 0777)
	if err != nil {
		return err
	}

	dest, err := os.Create(newFile)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		return err
	}

	return dest.Sync()
}

func (d DotfileDeployer) DeployCmd(dot string) error {
	for _, cmd := range d.conf.Deploy {
		expandedCmd := make([]string, 0, len(cmd))
		for _, arg := range cmd {
			expandedArg, err := d.expand(arg)
			if err != nil {
				return err
			}
			expandedCmd = append(expandedCmd, expandedArg)
		}

		cmdStr := strings.Join(expandedCmd, " ")
		origCmdStr := strings.Join(cmd, " ")
		if cmdStr != origCmdStr {
			cmdStr += " (expanded from " + origCmdStr + ")"
		}
		fmt.Println("Running command", cmdStr)

		if !d.dry {
			code, err := runCmd(expandedCmd)
			if err != nil {
				return err
			} else if code != 0 {
				errStr := fmt.Sprintf(
					"Command %s returned %d",
					cmdStr,
					code,
				)
				return errors.New(errStr)
			}

		}
	}

	return nil
}
