## medic completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(medic completion zsh)

To load completions for every new session, execute once:

#### Linux:

	medic completion zsh > "${fpath[1]}/_medic"

#### macOS:

	medic completion zsh > $(brew --prefix)/share/zsh/site-functions/_medic

You will need to start a new shell for this setup to take effect.


```
medic completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string      config file (default is $PWD/config.yaml)
      --grpc-host string   Server host (default "staging.stacklok.dev")
      --grpc-insecure      Allow establishing insecure connections
      --grpc-port int      Server port (default 443)
```

### SEE ALSO

* [medic completion](medic_completion.md)	 - Generate the autocompletion script for the specified shell

