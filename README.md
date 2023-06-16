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
simple package manager wrapper all in one executable. Different dotfile
configurations can be managed in different ways, and different programs can be
installed with different packages in different environments.

## Environment String

Estragon makes decisions based off of an environment string that's passed on the
initial run of the program. After this, the environment string can be specified
again in the command line, but you don't have to as Estragon stores it in a file
in the same directory as your `estragon.yaml`.

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
called `estragon.yaml`.

For future reference, "dotfile directory" means the same thing as "config
directory", that is a folder containing the dotfiles to actually be installed
for a particular program. It does _not_ refer to the directory containing the
`estragon.yaml` file, but any directory in _that_ folder is a dotfile directory.

The root level configurations are defined by this table:

| Key            | Description and possible values              |
| -------------- | -------------------------------------------- |
| `method`       | `"deep"`, `"shallow"`, `"copy"`, or `"none"` |
| `root`         | `"home"`, `"xdg"`, or a full path            |
| `dot-prefix`   | `true` or `false`                            |
| `environments` | Environment specific simple settings         |
| `packages`     | A [package specification map](#packages)     |
| `dots`         | A [dot map](#dots)                           |

### `method`

The `method` field specifies the default method by which the dotfiles will stay
in sync, by either copying the files directly (`"copy"`), creating a symlink to
every file (`"deep"`), creating as few symlinks as possible (`"shallow"`), or
not doing anything with the dotfiles at all (`"none"`). All of these will follow
the [rules of an individual dot](#rules). Different methods require different
frequencies of syncing if you are constantly modifying your configurations
(`"copy"`, `"deep"`, `"shallow"`, and `"none"` from most to least frequently
needing to sync).

### `root`

The `root` field specifies where the dotfiles' directory structures will be
placed. If it is `"home"`, it will place all files without [rules](#rules) in
your home directory. If it is `"xdg"`, it will use `$XDG_CONFIG_HOME/{dot}`
environment variable, or fall back to `$HOME/.config/:` on Unix-likes and
`$LOCALAPPDATA/:` on Windows (dollar sign is used for environment variables
regardless of operating system). If it is neither of these, it will assume that
the value is a full path, which processes environment variables. If the full
path contains `:`, it will replace that `:` with the name of the dotfile
directory.

Given a dotfile directory in the Estragon root called `dots` which contains
`dots/a` and `dots/dir/b`, the below table shows examples of how these
dotfiles will be installed on a Unix system.

| `root`         | `dots/a` location         | `dots/dir/b` location         |
| -------------- | ------------------------- | ----------------------------- |
| `"home"`       | `$HOME/a`                 | `$HOME/dir/b`                 |
| `"xdg"`        | `$XDG_CONFIG_HOME/dots/a` | `$XDG_CONFIG_HOME/dots/dir/b` |
| `"$HOME/conf"` | `$HOME/conf/a`            | `$HOME/conf/dir/b`            |
| `"/conf/:"`    | `/conf/dots/a`            | `/conf/dots/dir/b`            |

Remember to export `$XDG_CONFIG_HOME` manually if you are using `"xdg"` and
using Estragon to manage your shell configuration as those configs have probably
not be exported yet (`$LOCALAPPDATA` should be set already if you are on a
Windows machine)!

### `dot-prefix`

The `dot-prefix` field specifies if files with a name that starts with `dot-`
should have that prefix replaced with a lone dot (`.`). This is to avoid
problems that come with hidden files in editors and shells. It is `true` by
default. If it is `false`, no changes are made to files that start with `dot-`.
If a file is directly referenced by the [rules](#rules) (not by proxy by being
in a referenced folder), this setting is ignored.

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
the path to the file relative to the dotfile directory. The value is a path
(relative to the Estragon root directory or full) to the output file. If a value
is empty, it is not exported. If a key is empty, it acts as a wildcard for
unmatched files. If a key is a single slash (`"/"`), the value is a _folder_,
not a file, and specifies where the root is for the environment. The value
supports environment variables. Any files that are specified by the rules will
ignore the `root` and `dot-prefix` settings. See the [environment string
section] on how to format the environment key. For example:

```yaml
rules:
  "home":
    "gitconfig-home": "$HOME/.gitconfig"
    "": ""
  "work":
    "gitconfig-work": "$HOME/.gitconfig"
    "": ""
  "school":
    "gitconfig-school": "$HOME/.gitconfig"
    "": ""
```

#### Dot `packages`

The `packages` field maps package names to descriptions. The descriptions will
be printed while the packages are being installed. The keys are the package name
strings that will be expanded by the [`packages`](#packages) root-level
configuration.

[Go regexp syntax]: https://pkg.go.dev/regexp/syntax
