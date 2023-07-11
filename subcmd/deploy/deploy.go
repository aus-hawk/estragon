package deploy

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aus-hawk/estragon/config"
)

type DotManager interface {
	DotConfig(string) (config.DotConfig, error)
}

type FileDeployer interface {
	Copy(map[string]string) error
	Symlink(map[string]string) error
	Expand(string) string
}

type DotfileDeployer struct {
	mgr      DotManager
	deployer FileDeployer
	root     string
}

// NewDotfileDeployer creates a DotfileDeployer that gathers information from
// mgr and deploys them with the functions provided by deployer. The directory
// containing all of the dots is the root.
func NewDotfileDeployer(
	mgr DotManager,
	deployer FileDeployer,
	root string,
) DotfileDeployer {
	return DotfileDeployer{mgr, deployer, root}
}

// Deploy either copies or creates links of files within the dot file tree
// outside of that file tree. dot is the name of the dot that is being deployed.
// files is a slice of all of the files (not including directories) within the
// dot directory. dry determines if an action is actually performed (true) or if
// it will just be simulated by printing out what would actually happen (false).
func (d DotfileDeployer) Deploy(dot string, files []string, dry bool) error {
	dotConf, err := d.mgr.DotConfig(dot)
	if err != nil {
		return err
	}

	var fileMap map[string]string

	method := dotConf.Method

	switch method {
	case "deep", "copy":
		fileMap = d.deepCopyResolve(dotConf, files, dot)
	case "shallow":
		fileMap = d.shallowResolve(dotConf, files, dot)
	default:
		if method != "none" {
			errMsg := fmt.Sprintf("%s is not a valid method", method)
			err = errors.New(errMsg)
		}
		return err
	}

	expandedRoot := strings.Replace(dotConf.Root, "*", dot, 1)
	expandedRoot = d.deployer.Expand(expandedRoot)

	fmt.Println("Method:", dotConf.Method)
	fmt.Printf("Root: %s (expanded from %s)\n", expandedRoot, dotConf.Root)
	fmt.Println("Dot prefix:", dotConf.DotPrefix)
	fmt.Println("Rules:")
	for k, v := range dotConf.Rules {
		fmt.Printf("  %s -> %s\n", k, v)
	}

	switch method {
	case "deep", "shallow":
		fmt.Println("Creating the following symlinks (link -> original):")
		for dotfile, link := range fileMap {
			fmt.Printf("  %s -> %s\n", link, dotfile)
		}
		if !dry {
			err = d.deployer.Symlink(fileMap)
		}
	case "copy":
		fmt.Println("Copying the following files (original -> copy):")
		for dotfile, copyFile := range fileMap {
			fmt.Printf("  %s -> %s\n", dotfile, copyFile)
		}
		if !dry {
			err = d.deployer.Copy(fileMap)
		}
	}

	return err
}

func (d DotfileDeployer) deepCopyResolve(
	conf config.DotConfig,
	files []string,
	dot string,
) map[string]string {
	rules := conf.Rules

	// Ignore ruleless if an empty key exists in the rules map.
	_, ignoreRuleless := rules[""]
	unexpandedDots := make(map[string]string)

	for _, file := range files {
		outFile, ok := rules[file]
		if !ok {
			// Check if the file is in a subdirectory with a rule.
			outFile, ok = resolveSubdirRule(file, rules)
		}

		if ok {
			if outFile != "" {
				unexpandedDots[file] = outFile
			}
		} else if !ignoreRuleless {
			unexpandedDots[file] = ""
		}
	}

	return d.expandResolvedPaths(unexpandedDots, conf, dot, true)
}

func (d DotfileDeployer) shallowResolve(
	conf config.DotConfig,
	files []string,
	dot string,
) map[string]string {
	// All this function should do is filter the rules to only include real
	// files that exist according to the files variable.
	rules := conf.Rules

	unexpandedDots := make(map[string]string)
	for _, file := range files {
		outFile, ok := rules[file]
		if !ok {
			file, _, ok = splitSubdirRule(file, rules)
			outFile = rules[file]
		}

		if ok && outFile != "" {
			unexpandedDots[file] = outFile
		}
	}

	if len(unexpandedDots) == 0 {
		unexpandedDots[""] = ""
	}

	return d.expandResolvedPaths(unexpandedDots, conf, dot, false)
}

// resolveSubdirRule will take a file and find a subdirectory that is specified
// in the rules and use that rule to resolve the rule of the file. The bool it
// returns indicates if the resolution was successful.
//
// If the file is `a/b.txt`, and there is a rule from `a` to `c`, then the
// output file will be `c/b.txt`.
//
// Dot prefixes are expanded on the ruleless part of the path. So `a/dot-b/dot-d`
// turns into `c/.b/.d` using the last example's rules.
//
// If the closest parent directory with a rule maps to an empty string, an empty
// string is returned to indicate an ignored path.
func resolveSubdirRule(file string, rules map[string]string) (string, bool) {
	outDir, subpath, ok := splitSubdirRule(file, rules)

	if ok {
		outDir = rules[outDir]
		if outDir == "" {
			return "", true
		} else {
			subpath = expandDotPrefixes(subpath)
			return filepath.Join(outDir, subpath), true
		}
	} else {
		return "", false
	}
}

// splitSubdirRule splits the parts of the filepath that has a rule and the part
// that doesn't. A bool is also returned indicating if the split was done
// successfully.
func splitSubdirRule(file string, rules map[string]string) (string, string, bool) {
	dir := filepath.Dir(file)
	subpath := filepath.Base(file)

	// Loop until the directory matches or there are no more directories to
	// look at.
	_, ok := rules[dir]
	for !ok && filepath.Dir(dir) != "." {
		subpath = filepath.Join(filepath.Base(dir), subpath)
		dir = filepath.Dir(dir)
		_, ok = rules[dir]
	}

	return dir, subpath, ok
}

var dotPrefixRegexp *regexp.Regexp = regexp.MustCompile("(^|/)dot-")

// expandDotPrefix takes a path and replaces the "dot-" prefix of every part
// with a lone dot (".").
func expandDotPrefixes(path string) string {
	return dotPrefixRegexp.ReplaceAllString(path, "${1}.")
}

// expandResolvedPaths takes a fileMap and returns that map with full paths and
// expanded environment variables.
//
// Files that map to empty strings fall back to conf.Root, and have all of their
// dot prefixes expanded. All paths have their environment variables expanded.
func (d DotfileDeployer) expandResolvedPaths(
	fileMap map[string]string,
	conf config.DotConfig,
	dot string,
	expandPrefix bool,
) map[string]string {
	outRoot := strings.Replace(conf.Root, "*", dot, 1)
	dotRoot := filepath.Join(d.root, dot)

	expandedPaths := make(map[string]string)
	for file, outFile := range fileMap {
		if outFile == "" {
			// Empty output files default to the root.
			outBaseFile := file
			if expandPrefix && conf.DotPrefix {
				outBaseFile = expandDotPrefixes(outBaseFile)
			}
			outFile = filepath.Join(outRoot, outBaseFile)
		}
		file = filepath.Join(dotRoot, file)

		file = d.deployer.Expand(file)
		outFile = d.deployer.Expand(outFile)
		expandedPaths[file] = outFile
	}

	return expandedPaths
}
