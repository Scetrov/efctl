## efctl env extension

Manage the builder-scaffold extension flow

### Synopsis

The extension command groups operations defined in the EVE Frontier Builder Flow for Docker, such as init and publish.

### Options

```
  -h, --help   help for extension
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
  -w, --workspace string     Path to the workspace directory (default ".")
```

### SEE ALSO

* [efctl env](efctl_env.md)	 - Manage the local Sui development environment
* [efctl env extension init](efctl_env_extension_init.md)	 - Initialize the builder-scaffold by copying world artifacts
* [efctl env extension list](efctl_env_extension_list.md)	 - List all available extensions in the workspace
* [efctl env extension publish](efctl_env_extension_publish.md)	 - Publish a custom extension contract

