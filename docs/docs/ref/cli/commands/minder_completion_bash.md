---
title: minder completion bash
---
## minder completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(minder completion bash)

To load completions for every new session, execute once:

#### Linux:

	minder completion bash > /etc/bash_completion.d/minder

#### macOS:

	minder completion bash > $(brew --prefix)/etc/bash_completion.d/minder

You will need to start a new shell for this setup to take effect.


```
minder completion bash
```

### Options

```
  -h, --help              help for bash
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

