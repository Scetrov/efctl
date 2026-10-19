package setup

const (
	ScriptGenerateWorldEnv = "/workspace/builder-scaffold/docker/scripts/generate-world-env.sh"
	CmdDeployWorld         = `cd /workspace/world-contracts && \
  echo "execution boundary: sui-playground" && \
  echo "OS/architecture:" && { cat /etc/os-release 2>/dev/null || echo "OS information not found"; uname -m 2>/dev/null || true; } && \
  echo "pnpm launcher:" && (command -v pnpm || echo "pnpm launcher not found") && \
  echo "project packageManager:" && (grep '"packageManager"' package.json || echo "packageManager not found") && \
  echo "pnpm version:" && pnpm --version && \
  echo "pnpm-workspace.yaml:" && (cat pnpm-workspace.yaml || echo "pnpm-workspace.yaml not found") && \
  (pnpm approve-builds esbuild 2>/dev/null || true) && \
  pnpm install --prefer-offline && pnpm deploy-world`
	CmdConfigureWorld      = "cd /workspace/world-contracts && pnpm configure-world"
	CmdCreateTestResources = "cd /workspace/world-contracts && pnpm create-test-resources"
	FilePubLocalnetToml    = "Pub.localnet.toml"
	FilePubTestnetToml     = "Pub.testnet.toml"
)
