package dotfile

import (
	"path/filepath"
	"regexp"
)

// PathExpander is a type of function that takes a path and returns an expanded
// version of that path, as well as an error indicating if something went wrong
// expanding the path.
type PathExpander func(string) (string, error)

// Resolver is a type that is used to resolve files to maps from files to their
// output locations.
type Resolver struct {
	dotRoot, outRoot string
	dotPrefix        bool
	expand           PathExpander
}

// NewResolver creates a new Resolver. dotRoot is the root of all of the files
// that will be resolved. outRoot is the default location of the output files.
// dotPrefix indicates if "dot-" prefixes should be expanded. expand is a
// PathExpander that is used to expand the paths as they are being resolved.
func NewResolver(
	dotRoot string,
	outRoot string,
	dotPrefix bool,
	expand PathExpander,
) Resolver {
	return Resolver{dotRoot, outRoot, dotPrefix, expand}
}

// DeepResolve resolves the placement of files by mapping every input file to an
// output file and using the rules to change the location of individual files
// mentioned or all files within a mentioned folder, the more specific the
// folder the higher priority to take when resolving.
func (r Resolver) DeepResolve(
	files []string,
	rules map[string]string,
) (map[string]string, error) {
	// Ignore ruleless if an empty key exists in the rules map.
	_, ignoreRuleless := rules[""]
	fileMap := make(map[string]string)

	for _, file := range files {
		outFile, ok := rules[file]
		if !ok {
			// Check if the file is in a subdirectory with a rule.
			outFile, ok = resolveSubdirRule(file, rules, r.dotPrefix)
		}

		if ok {
			if outFile != "" {
				fileMap[file] = outFile
			}
		} else if !ignoreRuleless {
			fileMap[file] = ""
		}
	}

	return r.expandResolvedPaths(fileMap)
}

// ShallowResolve resolves the placement of files by creating a map from only
// files that exist and are in the rules to their associated location in the
// rules. Dot expansion is always ignored. If the rule map has no key-value
// pairs that match any of the files, the returned map has a single key from the
// root of the dots to the output root. If something goes wrong during file
// expansion, that is reflected in a non-nil error.
func (r Resolver) ShallowResolve(
	files []string,
	rules map[string]string,
) (map[string]string, error) {
	r.dotPrefix = false

	fileMap := make(map[string]string)
	for _, file := range files {
		outFile, ok := rules[file]
		if !ok {
			file, _, ok = splitSubdirRule(file, rules)
			outFile = rules[file]
		}

		if ok && outFile != "" {
			fileMap[file] = outFile
		}
	}

	if len(fileMap) == 0 {
		fileMap[""] = ""
	}

	return r.expandResolvedPaths(fileMap)
}

func (r Resolver) expandResolvedPaths(
	fileMap map[string]string,
) (map[string]string, error) {
	expandedFileMap := make(map[string]string)

	for file, outFile := range fileMap {
		if outFile == "" {
			// Empty output files default to the root.
			outBaseFile := file
			if r.dotPrefix {
				outBaseFile = expandDotPrefixes(outBaseFile)
			}
			outFile = filepath.Join(r.outRoot, outBaseFile)
		}
		file = filepath.Join(r.dotRoot, file)

		var err error
		file, err = r.expand(file)
		if err != nil {
			return nil, err
		}

		outFile, err = r.expand(outFile)
		if err != nil {
			return nil, err
		}

		expandedFileMap[file] = outFile
	}

	return expandedFileMap, nil
}

// resolveSubdirRule will take a file and find a subdirectory that is specified
// in the rules and use that rule to resolve the rule of the file. The bool it
// returns indicates if the resolution was successful.
//
// If the file is `a/b.txt`, and there is a rule from `a` to `c`, then the
// output file will be `c/b.txt`.
//
// If dot is true, dot prefixes are expanded on the ruleless part of the path.
// So `a/dot-b/dot-d` turns into `c/.b/.d` using the last example's rules.
//
// If the closest parent directory with a rule maps to an empty string, an empty
// string is returned to indicate an ignored path.
func resolveSubdirRule(
	file string,
	rules map[string]string,
	dotPrefix bool,
) (string, bool) {
	outDir, subpath, ok := splitSubdirRule(file, rules)

	if ok {
		outDir = rules[outDir]
		if outDir == "" {
			return "", true
		} else {
			if dotPrefix {
				subpath = expandDotPrefixes(subpath)
			}
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
