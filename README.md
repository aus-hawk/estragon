# Estragon

Estragon is a personal environment configuration program with emphasis on
simplicity and flexibility. It will keep track of your packages and sort out
your dotfiles so you can feel at home anywhere with one command.

## Why Estragon?

Tools like Ansible are extremely powerful, as those tools are made to command
fleets of computers to perform complex tasks. But with power comes complexity;
playbooks and role-based dotfile configurations can get very dense and are
difficult to get started with compared to dedicated dotfile managers because the
scope of Ansible and similar tools is so large.

On the other hand, dedicated dotfile managers are often too simplistic for
getting an environment set up from scratch. Instead, they focus on managing just
the files in your home directory and subdirectories, and not worrying about the
dependencies that those configurations have or files that live outside of your
home directory such as device configurations. Shells, programmable text editors,
window managers, and desktop environments often have configurations that rely on
something being installed beyond the base software that's being configured
directly.

Estragon provides a middle ground by being a powerful dotfile manager and a
simple package manager wrapped all into one executable. Different dotfile
configurations can be managed in different ways, and different programs can be
installed with different packages in different environments.

## Using Estragon

Estragon requires an [environment string](#environment-string) and an
[`estragon.yaml` configuration file](#estragonyaml-config-file-schema) in the
directory to install your packages and dotfiles. These are covered in detail in
their respective sections.

You can run `estragon help` to get more information about the subcommands and
the flags you can pass.

## Environment String

Estragon makes decisions based off of an environment string that's passed on the
initial run of the program with the `--env` flag. After this, the environment
string can be specified again in the command line, but you don't have to as
Estragon stores it in a directory in the same directory as your `estragon.yaml`.
You can pass a new environment string at it will replace the one already
associated with the directory.

The environment string is a series of fields that are space-separated. Some
settings in `estragon.yaml` make use of these fields to decide what values to
use. The environment string is matched against a series of space-separated
fields which the environment string must match all of, or it will not match. The
fields can use [Go regular expressions][Go regexp syntax], and some of the keys
may even be able to use the submatches when generating their values (in which
case, dollar signs are used and so literal dollar signs must be escaped with a
double dollar sign). An empty string is a wildcard and will match anything. If
there are multiple matches and one is the empty string wildcard, the
non-wildcard match will take precedence. If multiple non-wildcard matches exist,
it is not guaranteed which match will take precedence so take advantage of

Note that while these examples made use of `field` and `field:value` styled
fields, there is no restriction on what text can go in a field other than that
spaces are not allowed. You can format your fields however you want.

For example:

```yaml
"debian version:(.+) graphics-driver:(.+)": "driver-$2-$1"
```

This key will match any of the below environment strings:

```
debian version:stretch graphics-driver:gpu
version:buzz debian graphics-driver:company
laptop debian work version:jessie graphics-driver:cpu desktop:gnome
```

Notice that the order does not matter, and that only the fields specified are
checked. Respectively, the output string would be `driver-gpu-stretch`,
`driver-company-buzz`, and `driver-cpu-jessie` due to the `$1` and `$2` being
replaced by the first and second submatch in the key string.

### Negating the Environment String

If you want a certain combination of fields to cause a key to _not_ match, you
can separate the positive fields from negative false fields with a lone
exclamation mark (`!`). For example, a key of `windows.* laptop ! windows8 work`
will match all environment strings that have any field that starts with
`windows` and has the field `laptop`, but will fail if that field happens to be
`windows8` and the environment has the field `work`.

Note that the phrasing was "_and_ the environment has the field". If we wanted
the match to fail on either `windows8` _or_ `work`, we could change the key to
`windows.* laptop ! windows8|work`, which replaces the multiple conditions with
just a regular expression. Regular expression submatches are only considered in
the non-negated portions of the key, meaning submatches after the `!` are thrown
out.

All lone exclamation marks after the first are ignored.

## `estragon.yaml` Config File Schema

For Estragon to know what to do with a directory of dotfiles, it needs to be
configured with a file in the directory containing all of the dotfile folders
called `estragon.yaml`. It should be in the same directory as the one specified
(either the current one or the one passed with the `--dir` flag).

For future reference, "dotfile directory" means the same thing as "config
directory", that is a folder containing the dotfiles to actually be installed
for a particular program. It does _not_ refer to the directory containing the
`estragon.yaml` file, but any directory in _that_ folder is a dotfile directory.

The priority that the common configurations (`method`, `root`, and `dot-prefix`)
have, from highest to lowest is the following:

1. Dot level matching environment configuration
1. Dot level configuration
1. Global environment configuration
1. Global configuration

The root level configurations are defined by this table:

| Key            | Description and possible values                          |
| -------------- | -------------------------------------------------------- |
| `method`       | `"deep"`, `"shallow"`, `"copy"`, or `"none"`             |
| `root`         | A full path                                              |
| `check-cmd`    | An [environment-command map](#check-cmd-and-install-cmd) |
| `install-cmd`  | An [environment-command map](#check-cmd-and-install-cmd) |
| `dot-prefix`   | `true` or `false`                                        |
| `environments` | Environment specific simple settings                     |
| `packages`     | A [package specification map](#packages)                 |
| `dots`         | A [dot map](#dots)                                       |

### `method`

The `method` field specifies the default method by which the dotfiles will stay
in sync, by either copying every file directly (`"copy"`), creating a symlink to
every file (`"deep"`), creating symlinks only to specified files and folders in
[`rules`](#rules) (`"shallow"`), or not doing anything with the dotfiles at all
(`"none"`). All of these will follow the [rules of an individual dot](#rules).
Different methods require different frequencies of syncing if you are constantly
modifying your configurations (`"copy"`, `"deep"`, `"shallow"`, and `"none"`
from most to least frequently needing to sync).

If `method` is set to `"shallow"` and there are no rules, a single symlink is
created from the specified [`root`](#root) to the dot root.

Files will never be overwritten with a link or a copied file unless the file is
known to be owned by the Estragon directory.

### `root`

The `root` field specifies the full path to where the dotfiles' directory
structures will be placed. Environment variables are processed, as well as `~`
being expanded to `$HOME`. Environment variables should be denoted with the
typical dollar sign shell syntax regardless of OS. If an environment variable
does not exist, it will be treated as if it is an empty string. If the path
contains `*`, it will replace that `*` with the name of the dotfile directory,
which can be used to specify things like the XDG config directory if the
directory name matches the name of its config folder.

Given a dotfile directory in the Estragon root called `dots` which contains
`dots/a` and `dots/dir/b`, the below table shows examples of how these
dotfiles will be installed on a Unix system.

| `root`         | `dots/a` location | `dots/dir/b` location |
| -------------- | ----------------- | --------------------- |
| `"$HOME/conf"` | `$HOME/conf/a`    | `$HOME/conf/dir/b`    |
| `"~/conf"`     | `$HOME/conf/a`    | `$HOME/conf/dir/b`    |
| `"/conf/*"`    | `/conf/dots/a`    | `/conf/dots/dir/b`    |

### `check-cmd` and `install-cmd`

The `check-cmd` represents the command used to verify if a package is already
installed to your system. The `install-cmd` is the command that will actually
install the package if `check-cmd` indicates that it is not already. The actual
values of each of these keys are maps from environments to lists of strings, the
list representing the actual command to run. If the environment does not match
any of the keys, attempting to install packages with Estragon will fail.

The command should expect the name of the package to be the last argument. If
running the program with the package name returns a non-zero value, the
installation fails.

The commands are not run by any particular shell, so shell expansions will not
work.

For example, running Pacman to install a package is typically run as `pacman -S
[PACKAGE]`, and checking for a package with Pacman is `pacman -T [PACKAGE]`.
This means that if you use `arch` in your environment string to indicate any
Arch Linux-based distribution (which uses Pacman), you can set these arguments
like this:

```yaml
check-cmd:
  arch: ["pacman", "-T"]
install-cmd:
  arch: ["pacman", "-S"]
```

Note that these commands don't include things like `sudo`. If something like
`sudo` is needed to run your package manager, you should run Estragon itself
with a user-switching program.

If for some reason your package manager commands cannot accept the package name
as the last argument, you can write a shell script that's either in a directory
in your `PATH` or in your Estragon root directory and call it with
`./script.sh`.

### `dot-prefix`

The `dot-prefix` field specifies if files with a name that starts with `dot-`
should have that prefix replaced with a lone dot (`.`). This is to avoid
problems that come with hidden files in editors and shells. It is `true` by
default. If it is `false`, no changes are made to files that start with `dot-`.
If a file is directly referenced by the [rules](#rules) (not by proxy by being
in a referenced folder), or the dot is configured to use the shallow method,
this setting is ignored and treated as false for that particular file or the
entire folder respectively.

### `environments`

The `environments` field configures `method`, `root`, and `dot-prefix` by
environment. These configurations are put into their own category instead of
being configurable by environment themselves since they vary by environment
significantly less than things like packages and where to place dotfiles,
meaning they would often just be added noise. These configurations override the
defaults if the environment matches.

Example:

```yaml
environments:
  "windows.*":
    method: copy
    root: xdg
  bsd:
    method: shallow
```

### `packages`

The `packages` field allows you to create common names for different groups of
packages that exist across environments. The use case for this is when a package
has a different name across environments (e.g. `python`, `python3`, etc.), or
when the bootstrapping requires multiple packages that could be placed under a
single name (X11 setup, for example). When a package is referenced and matches
any of the package lists specified here, it is expanded to the list resolved to.
Otherwise, no expansion is done. The value is a map from environments to lists
of packages as strings. See the [environment string
section](#environment-string) for more information on environment string key
matching syntax. For example:

```yaml
packages:
  foo:
    debian: [bar, baz]
    ubuntu:
      - qux
      - quux
  foobar:
    "work|home": ["foobar3"]
    school: ["foobar3.0"]
    "debian-(bookworm|bullseye|jessie)": ["foobar-3"]
  dev-tools:
    "": [git, vim, go]
  driver:
    "driver:(abc|def|ghi)": ["driver-pkg-$1"]
  python:
    work: [python3]
    home: [python] # redundant and unnecessary
```

### `dots`

The `dots` field is used to specify how the dotfiles in a dotfile directory
should be installed, as well as what packages to install and why. It is a map
with keys being dotfile folder names. The values of each is a dot configuration.

Besides `packages`, the keys that have the same name as a global setting
overrides that setting. A dot configuration is defined by this table:

| Key            | Possible values                     |
| -------------- | ----------------------------------- |
| `method`       | See [`method`](#method)             |
| `root`         | See [`root`](#root)                 |
| `dot-prefix`   | See [`dot-prefix`](#dot-prefix)     |
| `environments` | See [`environments`](#environments) |
| `rules`        | A [rule configuration map](#rules)  |
| `packages`     | A [dot package map](#dot-packages)  |

#### `rules`

The `rules` field is a map from environment regular expressions to file maps. A
file map is the rules for where to place dotfiles. The key of each mapping is
the path to the file (or folder, but from here just refered to as files)
relative to the dotfile directory. If a key is empty, it acts as a wildcard for
unmatched files. Because only one location can be specified per file, an empty
key can have any value and all files that fall back to the wildcard are ignored
and therefore not deployed. If a value is the empty string, the associated key
will not be deployed. The value supports environment variables and `~` to
`$HOME` expansion. Any files that are specified by the rules will ignore the
`root` and `dot-prefix` settings. See the [environment string
section](#environment-string) on how to format the environment key.

For example:

```yaml
rules:
  "home":
    "gitconfig-home": "$HOME/.gitconfig" # can be shortened to "~/.gitconfig"
    "": ""
  "work":
    "gitconfig-work": "$HOME/.gitconfig"
    "": ""
  "school":
    "gitconfig-school": "$HOME/.gitconfig"
    "": ""
```

The method for resolving where a file will be placed using the deep or copy
methods is pretty simple:

1. If the file is any of the keys, the output file will be the value.
1. If the previous case isn't true but the file is within any of the directories
   specified as keys, the output file will be placed with the key directory
   replaced with the value directory. The most specific folder will take
   priority.
1. If the previous case isn't true and there is a wildcard to empty string (`"":
""`), then the file is ignored and there is no output file.
1. If none of the previous cases are true, the file is placed relative to the
   specified `root`.

As an example, say there is a file called `dot/dir/ect/f.txt`, where `dot` is a
folder that exists in the Estragon root:

1. If there is a key called `"dir/ect/f.txt"`, that key will take precedence and
   the value will be the one associated with this key.
1. If there is no key like the previous one but there is a `"dir/ect"` and a
   `"dir"` (although this is not recommended), the `"dir/ect"` will take
   precedence since it is more specific. If the value to that key is `"~/abc"`,
   then the output file will be `"~/abc/f.txt"`. If instead that value was
   associated with the `"dir"` key, the output file would have been
   `"~/abc/ect/f.txt"`.
1. If there is no key like any of the previous ones, but there is an empty key,
   Estragon acts as if `f.txt` did not exist.
1. If none of the previous conditions are true, then `f.txt` will be placed
   relative to the specified root. For example, if it was `"home"`, then the
   output file would be `"$HOME/dir/ect/f.txt"`.

#### Dot `packages`

The `packages` field maps package names to descriptions. The descriptions will
be printed while the packages are being installed. The keys are the package name
strings that will be expanded by the [`packages`](#packages) root-level
configuration.

[Go regexp syntax]: https://pkg.go.dev/regexp/syntax
