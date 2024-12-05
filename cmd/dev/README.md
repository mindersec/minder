# `mindev`

`mindev` is a command line utility to help you develop rules and profiles for
Minder.

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
mindev ruletype test -e /path/to/entity -p /path/to/profile -r /path/to/ruletype
```

`ruletype test` is intended for testing a single rule; the entity definition
must match the rule's `def.in_entity` type. The tested entity is defined as a
set of YAML properties in the entity file; for example, if you're testing a rule
type that's targetted towards a repository, the YAML must match the repository
schema.

e.g.

```yaml
github/repo_name: my-repo
github/repo_owner: my-org
github/repo_id: 123456789
github/clone_url: https://github.com/my-org/my-repo.git
is_private: false
is_fork: false
```

You can see complete entity schema in the [`examples`](https://github.com/mindersec/minder/tree/main/cmd/dev/examples) folder.

The profile is the path to the profile file. This is needed to test the rule
since rules often take definitions and parameters from the profile. Note that
the profile must instantiate the rule type you're testing.

The rule type is the path to the rule type file.

Finally, you will need to specify an auth token that can evaluate the
rule type, using the `TEST_AUTH_TOKEN` environment variable. You can
use [`gh` (the GitHub CLI)](https://github.com/cli/cli) to produce a
GitHub auth token. For example:

```bash
TEST_AUTH_TOKEN=$(gh auth token) mindev ruletype test -e /path/to/entity -p /path/to/profile -r /path/to/rule
```

### Linting a rule type

```bash
mindev ruletype lint -r /path/to/rule
```

This will give you basic validations on the rule type file.
