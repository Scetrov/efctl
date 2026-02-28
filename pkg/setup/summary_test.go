package setup

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── parseDeployLog ─────────────────────────────────────────────────

func TestParseDeployLog(t *testing.T) {
	input := `Starting deployment...
Pre-computed Character ID: 0xaabbccdd11
Pre-computed Character ID: 0xaabbccdd22
NWN Object Id: 0xee11ff2233
Storage Unit Object Id: 0xaa11bb2233
Storage Unit Object Id: 0xaa11bb2244
Gate Object Id: 0xcc44dd5566
Some other log line
`
	scanner := bufio.NewScanner(strings.NewReader(input))
	ids := parseDeployLog(scanner)

	assert.Equal(t, []string{"0xaabbccdd11", "0xaabbccdd22"}, ids.characters)
	assert.Equal(t, []string{"0xee11ff2233"}, ids.nwns)
	assert.Equal(t, []string{"0xaa11bb2233", "0xaa11bb2244"}, ids.ssus)
	assert.Equal(t, []string{"0xcc44dd5566"}, ids.gates)
}

func TestParseDeployLog_Empty(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(""))
	ids := parseDeployLog(scanner)
	assert.Empty(t, ids.characters)
	assert.Empty(t, ids.nwns)
	assert.Empty(t, ids.ssus)
	assert.Empty(t, ids.gates)
}

func TestParseDeployLog_NoMatches(t *testing.T) {
	input := "line 1\nline 2\nno hex ids here\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	ids := parseDeployLog(scanner)
	assert.Empty(t, ids.characters)
}

// ── parseEnvLog ────────────────────────────────────────────────────

func TestParseEnvLog(t *testing.T) {
	input := `ADMIN_ADDRESS=0xabc123def456
ADMIN_PRIVATE_KEY=suiprivkey0000fake0000
PLAYER_A_ADDRESS=0xa456def789
PLAYER_A_PRIVATE_KEY=suiprivkey1111fake1111
PLAYER_B_ADDRESS=0xbff9900aabb
PLAYER_B_PRIVATE_KEY=suiprivkey2222fake2222
SOME_OTHER=value
`
	scanner := bufio.NewScanner(strings.NewReader(input))
	env := parseEnvLog(scanner)

	assert.Equal(t, "0xabc123def456", env.adminAddress)
	assert.Equal(t, "suiprivkey0000fake0000", env.adminKey)
	assert.Equal(t, "0xa456def789", env.playerAAddress)
	assert.Equal(t, "suiprivkey1111fake1111", env.playerAKey)
	assert.Equal(t, "0xbff9900aabb", env.playerBAddress)
	assert.Equal(t, "suiprivkey2222fake2222", env.playerBKey)
}

func TestParseEnvLog_Empty(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(""))
	env := parseEnvLog(scanner)
	assert.Empty(t, env.adminAddress)
	assert.Empty(t, env.adminKey)
}

func TestParseEnvLog_Partial(t *testing.T) {
	input := "ADMIN_ADDRESS=0xaabbcc1122\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	env := parseEnvLog(scanner)
	assert.Equal(t, "0xaabbcc1122", env.adminAddress)
	assert.Empty(t, env.playerAAddress)
}

// ── buildOverrideYaml ──────────────────────────────────────────────

func TestBuildOverrideYaml_BothEnabled(t *testing.T) {
	yaml := buildOverrideYaml(true, true)
	assert.Contains(t, yaml, "postgres:")
	assert.Contains(t, yaml, "sui-pgdata:")
	assert.Contains(t, yaml, "frontend:")
	assert.Contains(t, yaml, "frontend-node-modules:")
	assert.True(t, strings.HasPrefix(yaml, "services:\n"))
}

func TestBuildOverrideYaml_GraphqlOnly(t *testing.T) {
	yaml := buildOverrideYaml(true, false)
	assert.Contains(t, yaml, "postgres:")
	assert.Contains(t, yaml, "sui-pgdata:")
	assert.NotContains(t, yaml, "frontend:")
	assert.NotContains(t, yaml, "frontend-node-modules:")
}

func TestBuildOverrideYaml_FrontendOnly(t *testing.T) {
	yaml := buildOverrideYaml(false, true)
	assert.NotContains(t, yaml, "postgres:")
	assert.Contains(t, yaml, "frontend:")
	assert.Contains(t, yaml, "frontend-node-modules:")
}

func TestBuildOverrideYaml_NoneEnabled(t *testing.T) {
	yaml := buildOverrideYaml(false, false)
	assert.Equal(t, "services:\n", yaml)
}

// ── graphqlServicesYaml / frontendServiceYaml ──────────────────────

func TestGraphqlServicesYaml(t *testing.T) {
	yaml := graphqlServicesYaml()
	assert.Contains(t, yaml, "postgres:")
	assert.Contains(t, yaml, "SUI_INDEXER_DB_URL")
	assert.Contains(t, yaml, "SUI_GRAPHQL_ENABLED")
	assert.Contains(t, yaml, "9125:9125")
}

func TestFrontendServiceYaml(t *testing.T) {
	yaml := frontendServiceYaml()
	assert.Contains(t, yaml, "frontend:")
	assert.Contains(t, yaml, "5173:5173")
	assert.Contains(t, yaml, "pnpm")
}

// ── overrideVolumesYaml ────────────────────────────────────────────

func TestOverrideVolumesYaml_Both(t *testing.T) {
	yaml := overrideVolumesYaml(true, true)
	assert.Contains(t, yaml, "sui-pgdata:")
	assert.Contains(t, yaml, "frontend-node-modules:")
}

func TestOverrideVolumesYaml_None(t *testing.T) {
	yaml := overrideVolumesYaml(false, false)
	assert.Empty(t, yaml)
}

// ── patchEntrypointPostgresWait ────────────────────────────────────

func TestPatchEntrypointPostgresWait_InjectsWaitBlock(t *testing.T) {
	content := "#!/bin/bash\n# ---------- start local node ----------\nsui start"
	result := patchEntrypointPostgresWait(content)
	assert.Contains(t, result, "wait for postgres")
	assert.Contains(t, result, "pg_isready")
}

func TestPatchEntrypointPostgresWait_Idempotent(t *testing.T) {
	content := "#!/bin/bash\n# ---------- start local node ----------\nsui start"
	first := patchEntrypointPostgresWait(content)
	second := patchEntrypointPostgresWait(first)
	assert.Equal(t, first, second)
}

// ── patchEntrypointSuiStart ────────────────────────────────────────

func TestPatchEntrypointSuiStart_InjectsArgs(t *testing.T) {
	content := "sui start --with-faucet --force-regenesis &"
	result := patchEntrypointSuiStart(content)
	assert.Contains(t, result, "SUI_START_ARGS")
	assert.Contains(t, result, "--with-graphql")
}

func TestPatchEntrypointSuiStart_Idempotent(t *testing.T) {
	content := "sui start --with-faucet --force-regenesis &"
	first := patchEntrypointSuiStart(content)
	second := patchEntrypointSuiStart(first)
	assert.Equal(t, first, second)
}

// ── patchEntrypointLoopTimings ─────────────────────────────────────

func TestPatchEntrypointLoopTimings_Updates30to60(t *testing.T) {
	content := `for i in $(seq 1 30); do
  if [ "$i" -eq 30 ]; then
    echo "fail"
  fi
done`
	result := patchEntrypointLoopTimings(content)
	assert.Contains(t, result, "seq 1 60")
	assert.Contains(t, result, `-eq 60`)
	assert.NotContains(t, result, "seq 1 30")
}

func TestPatchEntrypointLoopTimings_Idempotent(t *testing.T) {
	content := `for i in $(seq 1 30); do
  if [ "$i" -eq 30 ]; then`
	first := patchEntrypointLoopTimings(content)
	second := patchEntrypointLoopTimings(first)
	assert.Equal(t, first, second)
}

// ── orchestration with mocks ───────────────────────────────────────

func TestCleanEnvironment_CallsCleanup(t *testing.T) {
	mock := new(mockContainerClient)
	mock.On("Cleanup").Return(nil)

	err := CleanEnvironment(mock)
	require.NoError(t, err)
	mock.AssertExpectations(t)
}

func TestCleanEnvironment_PropagatesError(t *testing.T) {
	mock := new(mockContainerClient)
	mock.On("Cleanup").Return(assert.AnError)

	err := CleanEnvironment(mock)
	assert.Error(t, err)
}

// ── CloneRepositories with mocks ───────────────────────────────────

func TestCloneRepositories_Success(t *testing.T) {
	g := new(mockGitClient)
	ws := t.TempDir()

	g.On("SetupWorkDir", ws).Return(nil)
	g.On("CloneRepository", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	g.On("CheckoutBranch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	err := CloneRepositories(g, ws)
	require.NoError(t, err)
	g.AssertExpectations(t)
	// Should have cloned two repos (world-contracts + builder-scaffold)
	g.AssertNumberOfCalls(t, "CloneRepository", 2)
	g.AssertNumberOfCalls(t, "CheckoutBranch", 2)
}

func TestCloneRepositories_SetupFails(t *testing.T) {
	g := new(mockGitClient)
	g.On("SetupWorkDir", mock.Anything).Return(assert.AnError)

	err := CloneRepositories(g, "/tmp/fail")
	assert.Error(t, err)
}

func TestCloneRepositories_CloneFails(t *testing.T) {
	g := new(mockGitClient)
	ws := t.TempDir()
	g.On("SetupWorkDir", ws).Return(nil)
	g.On("CloneRepository", mock.Anything, mock.Anything).Return(assert.AnError)

	err := CloneRepositories(g, ws)
	assert.Error(t, err)
}
