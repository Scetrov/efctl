## efctl doctor

Print diagnostic information about the environment

### Synopsis

Prints a non-destructive summary of the local environment useful for debugging
and bug reports, including: efctl version, OS details, container runtime, Node.js,
git, the state of running containers, port availability, and the git ref of any
checked-out builder-scaffold and world-contracts repositories.

Also reports source-labelled CPU features, runtime client/server platforms, actual
Sui image identity, and host/container Sui versions. When the managed Sui container
is stopped or absent, doctor may create a transient isolated probe of an existing
local immutable image. It never pulls images, starts the managed environment,
initializes a Sui client, or mounts host/managed data. Probes override the entrypoint,
disable networking, use a read-only root and resource limits, and are cleaned up.
Unsupported isolation or image-declared volumes cause probing to be skipped.

New probes share a 20-second budget with five seconds per command and up to three
additional seconds for cleanup; existing report gatherers are not covered by this
budget. Local, remote/VM server, image, and execution-time CPU views are distinct.
Unavailable fields include reasons. Version success is not proof of node health;
architecture mismatch suggests possible emulation, not a confirmed crash cause.
Update efctl, run doctor from your workspace, and review the report before sharing
it with support. A cleanup failure identifies the probe to inspect/remove.

--crash is opt-in and off by default. Default doctor does not query systemd,
coredumpctl, journalctl, or a debugger, and it does not add a crash section.
With --crash, doctor appends container exit/log hints and, only on a proven-local
Linux kernel, field-limited host crash metadata. It does not enable core dumps,
extract a core, print crash environment or maps, install helpers, or change the
doctor exit status when crash evidence is unavailable. Crash collection has its
own 20-second budget, five seconds per metadata command, and eight seconds for
one debugger invocation. Review that section before sharing it; do not attach
core files.

```
efctl doctor [flags]
```

### Options

```
      --crash              Opt-in local crash correlation; review before sharing and do not attach core files
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

