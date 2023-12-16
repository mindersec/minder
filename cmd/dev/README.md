# `mindev`

`mindev` is a command line utility to help you develop rules and profiles for Minder.

## Building

From the `minder` root directory, run:

```bash
make build
```

## Usage

```bash
mindev [command]
```

### Testing a rule type

```bash
mindev ruletype test -e /path/to/entity -p /path/to/profile -r /path/to/rule
```

The entity is the path to the entity file, in case you're testing a rule type
that's targetted towards a repository, the YAML must match the repository
schema.

e.g. 

```yaml
name: my-repo
owner: my-org
repo_id: 123456789
clone_url: https://github.com/my-org/my-repo.git
```

The profile is the path to the profile file. This is needed to test the rule
since rules often take definitions and parameters from the profile. Note that
the profile must instantiate the rule type you're testing.

Finally, the rule type is the path to the rule type file.

### Linting a rule type

```bash
mindev ruletype lint -r /path/to/rule
```

This will give you basic validations on the rule type file.
