---
title: minder completion zsh
---
## minder completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(minder completion zsh)

To load completions for every new session, execute once:

#### Linux:

	minder completion zsh > "${fpath[1]}/_minder"

#### macOS:

	minder completion zsh > $(brew --prefix)/share/zsh/site-functions/_minder

You will need to start a new shell for this setup to take effect.


```
minder completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string            Config file (default is $PWD/config.yaml)
      --grpc-host string         Server host (default "api.stacklok.com")
      --grpc-insecure            Allow establishing insecure connections
      --grpc-port int            Server port (default 443)
      --identity-client string   Identity server client ID (default "minder-cli")
      --identity-realm string    Identity server realm (default "stacklok")
      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")
```

### SEE ALSO

* [minder completion](minder_completion.md)	 - Generate the autocompletion script for the specified shell

