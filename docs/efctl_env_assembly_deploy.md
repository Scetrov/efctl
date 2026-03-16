## efctl env assembly deploy

Deploy a new assembly

### Options

```
  -h, --help                   help for deploy
      --item-id uint           Unique Item ID for the assembly
      --location-hash string   Location hash (hex) (default "0x0000000000000000000000000000000000000000000000000000000000000000")
      --on-behalf-of string    Character alias or ID (optional)
      --online                 Automatically online the assembly after deployment
      --type-id uint           Type ID for the assembly
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
  -w, --workspace string     Path to the workspace directory (default ".")
```

### SEE ALSO

* [efctl env assembly](efctl_env_assembly.md)	 - Manage Smart Assemblies
* [efctl env assembly deploy gate](efctl_env_assembly_deploy_gate.md)	 - Deploy a Smart Gate
* [efctl env assembly deploy storage](efctl_env_assembly_deploy_storage.md)	 - Deploy a Smart Storage Unit
* [efctl env assembly deploy turret](efctl_env_assembly_deploy_turret.md)	 - Deploy a Smart Turret

