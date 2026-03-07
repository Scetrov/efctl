## efctl env

Manage the local Sui development environment

### Synopsis

The env command groups operations to bring up and tear down the EVE Frontier local development environment.

### Options

```
  -h, --help               help for env
  -w, --workspace string   Path to the workspace directory (default ".")
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
```

### SEE ALSO

* [efctl](efctl.md)	 - efctl manages the local EVE Frontier Sui development environment
* [efctl env dash](efctl_env_dash.md)	 - Launch the environment dashboard
* [efctl env down](efctl_env_down.md)	 - Tear down the local environment
* [efctl env extension](efctl_env_extension.md)	 - Manage the builder-scaffold extension flow
* [efctl env run](efctl_env_run.md)	 - Run a script in the builder-scaffold container
* [efctl env shell](efctl_env_shell.md)	 - Open a shell inside the running container
* [efctl env status](efctl_env_status.md)	 - Show environment status without launching the dashboard
* [efctl env up](efctl_env_up.md)	 - Bring up the local environment

