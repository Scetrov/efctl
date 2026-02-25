package setup

const (
	ScriptGenerateWorldEnv = "/workspace/builder-scaffold/docker/scripts/generate-world-env.sh"
	CmdDeployWorld         = "cd /workspace/world-contracts && pnpm install && pnpm deploy-world"
	CmdConfigureWorld      = "cd /workspace/world-contracts && pnpm configure-world"
	CmdCreateTestResources = "cd /workspace/world-contracts && pnpm create-test-resources"
	FilePubLocalnetToml    = "Pub.localnet.toml"
	FilePubTestnetToml     = "Pub.testnet.toml"
)
