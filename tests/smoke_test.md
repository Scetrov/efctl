# efctl Smoke Test Runbook

Manual runbook for smoke-testing the `efctl` CLI end-to-end.
Run from the root of this repository unless otherwise stated.

---

## Prerequisites

Ensure the following are installed and available on your `$PATH`:

| Tool               | Min version            | Check                                    |
| ------------------ | ---------------------- | ---------------------------------------- |
| Docker or Podman   | Docker 24+ / Podman 4+ | `docker --version` or `podman --version` |
| Git                | any                    | `git --version`                          |
| Node.js            | ≥ 20                   | `node --version`                         |
| Sui CLI (optional) | any                    | `sui --version`                          |

---

## 1. Build

```bash
go build -o output/efctl main.go
export PATH="$(pwd)/output:$PATH"
```

**Expected:** No errors. `efctl --version` prints the version string.

```bash
efctl --version
```

---

## 2. Create an isolated workspace

```bash
mkdir -p smoke-test
cd smoke-test
```

All subsequent commands are run from inside `smoke-test/`.

---

## 3. `efctl env up` — Bring up the full environment

```bash
efctl env up
```

**Expected output (in order):**

- ✅ Prerequisites checked (Node, Docker/Podman, Git, Port 9000 free)
- ✅ Clones `world-contracts` and `builder-scaffold` into the workspace
- ✅ Builds and starts the `sui-playground` container
- ✅ Deploys world contracts (several `SUCCESS Execution complete` lines)
- ✅ Spawns game structures
- ✅ Prints a deployment summary table with `WORLD_PACKAGE_ID`, `ADMIN_ADDRESS`, etc.
- ✅ Final line: `🌍 Environment is up!`

**Verify the container is running:**

```bash
docker ps --filter name=sui-playground
# or
podman ps --filter name=sui-playground
```

**Expected:** `sui-playground` is listed and in `Up` state.

---

## 4. `efctl env extension init` — Initialise builder-scaffold

```bash
efctl env extension init
```

**Expected output:**

- ✅ `Copied world artifacts into builder-scaffold deployments.`
- ✅ `Configured builder-scaffold .env.`
- ✅ `builder-scaffold successfully initialized.`

**Verify the artefacts were copied:**

```bash
ls builder-scaffold/deployments/localnet/
# Should contain: extracted-object-ids.json and possibly Pub.localnet.toml

cat builder-scaffold/.env | grep -E "WORLD_PACKAGE_ID|ADMIN_ADDRESS"
# Should show non-empty values
```

---

## 5. `efctl env extension publish` — Publish an extension contract

```bash
efctl env extension publish smart_gate
```

**Expected output:**

- ✅ `Executing publish inside container at /workspace/builder-scaffold/move-contracts/smart_gate...`
- ✅ `INCLUDING DEPENDENCY MoveStdlib` / `Sui` / `World`
- ✅ `BUILDING smart_gate`
- ✅ A JSON blob from the Sui CLI
- ✅ `BUILDER_PACKAGE_ID = 0x...`
- ✅ `EXTENSION_CONFIG_ID = 0x...` _(if the contract creates an ExtensionConfig object)_
- ✅ `builder-scaffold/.env updated with published IDs.`
- ✅ `Extension contract published successfully.`

**Verify idempotency — run it a second time:**

```bash
efctl env extension publish smart_gate
```

**Expected:** Same success output. Should **not** fail with _"Your package is already published"_.

**Verify .env was updated:**

```bash
cat builder-scaffold/.env | grep -E "BUILDER_PACKAGE_ID|EXTENSION_CONFIG_ID"
# Both should have 0x... values
```

---

## 6. `efctl env run` — Run a script inside the container

```bash
efctl env run "sui client active-address"
```

**Expected:** Prints the active Sui address (the ef-admin address).

```bash
efctl env run "sui client envs"
```

**Expected:** Shows `ef-localnet` as an available environment.

---

## 7. `efctl graphql object` — Query a deployed object

Pick any `OBJECT_ID` from the deployment summary printed in step 3.

```bash
efctl graphql object <OBJECT_ID>
```

**Expected:** JSON response describing the object from the local GraphQL endpoint.

---

## 8. `efctl env down` — Tear down the environment

```bash
efctl env down
```

**Expected output:**

- ✅ Container `sui-playground` stopped and removed
- ✅ Docker images removed
- ✅ Volumes removed

**Verify:**

```bash
docker ps --filter name=sui-playground
# Should return empty
```

---

## 9. Clean up the workspace

```bash
cd ..
rm -rf smoke-test
```

---

## Pass criteria

| Step           | Command                                  | Pass condition                                     |
| -------------- | ---------------------------------------- | -------------------------------------------------- |
| 2              | `efctl --version`                        | Prints version                                     |
| 3              | `efctl env up`                           | `🌍 Environment is up!` line present               |
| 4              | `efctl env extension init`               | `builder-scaffold successfully initialized.`       |
| 5 (first run)  | `efctl env extension publish smart_gate` | `Extension contract published successfully.`       |
| 5 (second run) | `efctl env extension publish smart_gate` | Same success, no "already published" error         |
| 5 (.env)       | `cat builder-scaffold/.env`              | `BUILDER_PACKAGE_ID` and `EXTENSION_CONFIG_ID` set |
| 6              | `efctl env run ...`                      | Output from inside container                       |
| 7              | `efctl graphql object ...`               | JSON object response                               |
| 8              | `efctl env down`                         | Container removed                                  |
