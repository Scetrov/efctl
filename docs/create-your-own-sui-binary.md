# Create Your Own Sui Binary

This guide was written for how to build a native Sui executable for an WSL2 Linux environment in Windows. For those who experianced a `SIGILL` & illegal instruction error and are using the "efctl" community tool

Please read the entire guide once before you start going through all the steps.

## 1. Check your CPU flags

If you experianced a `SIGILL` or illegal instruction error, use `efctl doctor` to check your CPU flags, this allows you to check the CPU flags exposed to the WSL & Linux environment, see if your getting a warning, or if the flags include `ADX` and `BMI2`

Some x86-64 Sui builds or dependencies (including `blst`) can use ADX/BMI2 instructions unavailable on older CPUs. Missing flags are a compatibility hint, not proof of a SIGILL cause. ARM CPUs do not require these x86 instructions; keep local and container/remote-runtime CPU evidence separate.

You can also check in WSL with

```
uname -m
grep -m1 '^flags' /proc/cpuinfo
```

## 2. Check your environment

Install the common native build prerequisites and Rust

```
sudo apt update
sudo apt install -y \
  build-essential clang cmake pkg-config \
  libssl-dev libclang-dev protobuf-compiler git curl
```

Sui revisions can require additional libraries or specific toolchain versions. Follow the prerequisites documented by the selected Sui revision.

## 3. Select the compatible Sui source revision

Use the Sui repository and revision required by your EVE Frontier development environment. Confirm the version expected by the project before building.

For the upstream MystenLabs repository, a checkout can begin with:

```
mkdir -p ~/src
cd ~/src
git clone https://github.com/MystenLabs/sui.git
cd sui
```

Check out the specific compatible tag or commit once it has been identified:

```
git checkout <COMPATIBLE_SUI_TAG_OR_COMMIT>
```

Replace the placeholder with the actual tag or commit. Do not build from an arbitrary branch.

Inspect the repository's `rust-toolchain.toml` or `rust-toolchain` file and follow its required Rust toolchain.

**Important:** Build a Sui revision compatible with the EVE Frontier environment and its Move packages. It needs to match the version used by Frontier.

## 4. Configure Rust compilation

From the root of the Sui source checkout, configure by disabling ADX and BMI2 as compiler target features.
This guide used the `x86_64` configuration. But you can also build native

```
export RUSTFLAGS="-C target-cpu=x86-64 -C target-feature=-bmi2,-adx"
```

These settings request compatible code generation; they do not by themselves prove that every dependency or runtime code path avoids those instructions. Inspect the selected revision's Cargo configuration and build scripts for flags that might override these settings.

```
cargo build --release --bin sui
```

You can also build native if your using it for your specific device

```
RUSTFLAGS="-C target-cpu=native" cargo build --release --bin sui
```

After a successful build, the executable is commonly located at:

```text
target/release/sui
```

Confirm the file exists and runs:

```bash
./target/release/sui --version
```

Keep a record of the source commit, Rust toolchain version, compiler flags, and output version alongside the build.

## 5 Test The Sui binary

```
./target/release/sui start --force-regenesis
```

`Ctrl C` if you get the first epoch starts

## 6. Use The Sui binary with efctl

Create a backup copy of your `sui` binary

put the sui binary in the `docker/` folder in efctl/builder-scaffold/docker/`

** Important: ** Make sure efctl has already cloned the builder-scaffold repository in itself before you put in in the `docker/` folder, keep a copy of the `Sui` binary so you don't lose it if the repository reference is reloaded.

If you want to use your `Sui` binary without efctl in your own environment go to the bottom of the page, to setup correctly.

## 7. Start the enviornment

Make sure to delete the old image. And close the previous environment with `efctl env down`
run `efctl env up`, and `efctl env down` if you haven't already.

The builder Dockerfile expects the custom executable in the `docker` folder (just make a copy of your Sui binary in there)

```
builder-scaffold/docker/sui
```

efctl will select a non-empty regular `docker/sui` file without executing it during preparation. A symlink, empty file, unhydrated Git LFS pointer, or unsupported Dockerfile layout produces an error before modifying the Dockerfile or entrypoint. For an LFS-backed artifact, run `git lfs pull` inside `builder-scaffold` first.

These input checks do not authenticate the executable or guarantee architecture/library compatibility. Use trusted source and retain the build provenance described above. You can start your environment with `efctl env up`.

**Important:** Changes in `builder-scaffold/docker/` will be lost if you change the repository references in `efctl.yaml`, keep a duplicate of your `sui` binary elsewhere so you don't lose it

### Good now you can become a Builder on EVE Frontier.

## Troubleshooting (with efctl)

**The build fails because of Rust or dependency versions:** Follow the toolchain and dependency instructions for the exact Sui source revision. (The Sui binary version needs to match the version used by eve frontier)

**The binary works but not in efctl:** make sure the previous efctl enviorment was closed with `efctl env down`.
* Check Docker-Desktop or your Docker-Engine, and make your old containers and images have been deleted.

**The binary still fails with an illegal instruction:** Make sure to put a copy of your compiled `sui` binary in the `efctl/builder-scaffold/docker` folder and run (efctl env up)
* Test the binary manually following the EVE Frontier Builders Guide Docs "https://docs.evefrontier.com/quickstart/environment-setup" But use your custom Sui binary in the builder-scaffold instead of suiup.
* Review the compiler flags and rebuild the affected components as needed. Check whether a dependency, build script, precompiled library, or runtime-selected code path uses unsupported instructions (May require you to trace the SIGILL failure and illegal intruction)
* If your still getting `SIGILL` Errors, and your CPU supports the ADX & BMI2 flags, you can create an open an Issue on GitHub.

# Setting up your Sui binary in your own builder-scaffold clone/fork

## 1. Adjust Your `Dockerfile`

The EVE Frontier builder scaffold can use a custom executable. You can do this by putting a `sui` binary in the `builder-scaffold/docker/` directory
commenting out the Sui CLI installation block (this will exist in you `efctl/` clone)

```
# Original Sui CLI installation via suiup — retained but disabled.
# ENV PATH="/root/.local/bin:/root/.suiup/bin:${PATH}"
# RUN curl -sSfL https://raw.githubusercontent.com/MystenLabs/suiup/main/install.sh | sh \
#     && echo y | suiup install sui \
#     && sui --version
```
Then setting `sui` in the `Dockerfile`
```
COPY sui /usr/local/bin/sui
RUN chmod +x /usr/local/bin/sui \
    && sui --version
```

Thank you for reading the entire guide.

## 2. Install the executable into the builder scaffold

The builder Dockerfile expects the custom executable in the `docker` folder (just make a copy of your Sui binary in there)

```
builder-scaffold/docker/
```

efctl will handle it automatically once its in the `docker/` folder

you can also use it manually with your own builder-scaffold clone,

## 8. Rebuild the Docker image

A custom executable on the WSL host does not automatically replace the executable inside an existing container image. (thats what the Sui binary is put in, the "image")

In (Prefered) Docker Engine Or Docker Desktop, make sure to delete the old image and stop any running containers

* Important: * From here you can follow the EVE Frontier Docs on the website.

Thanks for reding the entire guide.