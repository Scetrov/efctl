## efctl doctor

Print diagnostic information about the environment

### Synopsis

Prints a non-destructive summary of the local environment useful for debugging
and bug reports, including: efctl version, OS details, container runtime, Node.js,
git, the state of running containers, port availability, and the git ref of any
checked-out builder-scaffold and world-contracts repositories.

```
efctl doctor [flags]
```

### Options

```
  -h, --help               help for doctor
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

