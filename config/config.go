package config

import (
	"errors"
	"fmt"
	"sort"

	"github.com/aus-hawk/estragon/env"
	"gopkg.in/yaml.v3"
)

type schema struct {
	Common       common              `yaml:",inline"`
	CheckCmd     map[string][]string `yaml:"check-cmd"`
	InstallCmd   map[string][]string `yaml:"install-cmd"`
	Validate     map[string][]string
	Environments map[string]common
	Packages     map[string]map[string][]string
	Dots         map[string]dot
}

type dot struct {
	Common       common `yaml:",inline"`
	Environments map[string]common
	Rules        map[string]map[string]string
	Deploy       map[string][][]string
	Packages     map[string]string
}

type common struct {
	Method    string
	Root      string
	DotPrefix *bool `yaml:"dot-prefix,omitempty"`
}

type EnvSelector interface {
	Select(keys []string) (key string, fields []string)
	Matches(string) bool
}

// A Config contains all of the information about how files and packages are
// installed.
type Config struct {
	schema   schema
	selector EnvSelector
}

// NewConfig creates a config based on the contents of the YAML content `in`,
// and the Environment `e`. If something goes wrong parsing `in`, err is
// non-nil.
func NewConfig(in []byte, s EnvSelector) (c Config, err error) {
	err = yaml.Unmarshal(in, &c.schema)
	c.selector = s
	return
}

// AllDots returns the slice of all dots that are defined within the config.
func (c Config) AllDots() []string {
	d := make([]string, 0, len(c.schema.Dots))
	for k := range c.schema.Dots {
		d = append(d, k)
	}
	sort.Strings(d)
	return d
}

// ValidateEnv checks the environment string against each validation set in the
// configuration. It returns non-nil if validation fails.
func (c Config) ValidateEnv() error {
	for k, v := range c.schema.Validate {
		if c.selector.Matches(k) && !c.envMatchesAny(v) {
			return fmt.Errorf(
				`No match for environment found in validate key "%s"`,
				k,
			)
		}
	}

	return nil
}

func (c Config) envMatchesAny(l []string) bool {
	for _, e := range l {
		if c.selector.Matches(e) {
			return true
		}
	}
	return false
}

// CheckCmd returns the check command that matches the environment. If none of
// the environments match, nil is returned.
func (c Config) CheckCmd() []string {
	checkers := c.schema.CheckCmd
	key, _ := c.selector.Select(mapKeys(checkers))
	cmd, _ := checkers[key]
	return cmd
}

// InstallCmd returns the install command that matches the environment. If none
// of the environments match, nil is returned.
func (c Config) InstallCmd() []string {
	installers := c.schema.InstallCmd
	key, _ := c.selector.Select(mapKeys(installers))
	cmd, _ := installers[key]
	return cmd
}

// A Package represents a single key-value pair in the package map of a dot. The
// Name is the key of a package a dot is using. The Desc is the description of
// the purpose the specified package serves for the dot. The List is the list of
// expanded package names if the package is specified in the global package map.
type Package struct {
	Name string
	Desc string
	List []string
}

// Packages gets a dot `dotName`'s list of packages and their associated
// expansions as a slice. If the dot specified does not exist, a non-nil error
// will be the second returned value.
func (c Config) Packages(dotName string) ([]Package, error) {
	dot, ok := c.schema.Dots[dotName]
	if !ok {
		return nil, errors.New("Dot " + dotName + " not in config")
	}

	packages := make([]Package, 0, len(dot.Packages))
	for name, desc := range dot.Packages {
		pkg := Package{
			Name: name,
			Desc: desc,
			List: c.expandPackage(name),
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (c Config) expandPackage(pkgName string) []string {
	packages, ok := c.schema.Packages[pkgName]
	if !ok {
		// The package is not expanded in any way.
		return []string{pkgName}
	}

	key, fields := c.selector.Select(mapKeys(packages))

	pkgList, ok := packages[key]
	if !ok {
		// The package is not expanded under the current environment.
		return []string{pkgName}
	}

	match := env.NewMatch(key, fields)
	realPkgs := make([]string, 0, len(pkgList))
	for _, pkg := range pkgList {
		realPkg := match.Replace(pkg)
		realPkgs = append(realPkgs, realPkg)
	}

	return realPkgs
}

// A DotConfig holds the most specific settings that can be applied to the
// installation of dotfiles as specified by a dot configuration in the config.
// The Rules are set according to the configuration and the environment, and are
// exclusive to the dot it is a config of. The Method, Root, and DotPrefix are
// values that can be set in multiple places, with the more specific
// configurations taking precedence over the more general ones.
//
// From specific to general: environment settings within the dot config, the
// values of the dot config itself, the environment settings of the global
// config, the values of the global config itself.
type DotConfig struct {
	Method       string
	Root         string
	DotPrefix    bool
	dotPrefixSet bool
	Rules        map[string]string
	Deploy       [][]string
}

// Get the DotConfig associated with a specific dot. If the dot does not exist
// in the config, it has all of the globally specified defaults.
func (c Config) DotConfig(dotName string) (d DotConfig) {
	// The zero value is fine to use.
	dot := c.schema.Dots[dotName]

	// Rules are only set in one place.
	key, fields := c.selector.Select(mapKeys(dot.Rules))
	envRules, ok := dot.Rules[key]
	if ok {
		templatedRules := make(map[string]string)
		match := env.NewMatch(key, fields)
		for k, v := range envRules {
			k = match.Replace(k)
			templatedRules[k] = v
		}
		d.Rules = templatedRules
	}

	// Deploy commands are also only set in one place.
	key, _ = c.selector.Select(mapKeys(dot.Deploy))
	envDeploy, ok := dot.Deploy[key]
	if ok {
		d.Deploy = envDeploy
	}

	// Apply common config from dot-specific environment settings.
	key, _ = c.selector.Select(mapKeys(dot.Environments))
	commonConf, ok := dot.Environments[key]
	if ok {
		d = applyCommonDotConfig(commonConf, d)
	}

	// Apply common config from dot settings.
	d = applyCommonDotConfig(dot.Common, d)

	// Apply common config from environment-specific global settings.
	key, _ = c.selector.Select(mapKeys(c.schema.Environments))
	commonConf, ok = c.schema.Environments[key]
	if ok {
		d = applyCommonDotConfig(commonConf, d)
	}

	// Apply common config from global settings.
	d = applyCommonDotConfig(c.schema.Common, d)

	if !d.dotPrefixSet {
		d.DotPrefix = true
	}

	return
}

func applyCommonDotConfig(c common, d DotConfig) DotConfig {
	if d.Method == "" {
		d.Method = c.Method
	}
	if d.Root == "" {
		d.Root = c.Root
	}
	if !d.dotPrefixSet && c.DotPrefix != nil {
		d.DotPrefix = *c.DotPrefix
		d.dotPrefixSet = true
	}
	return d
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	ks := make([]K, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
