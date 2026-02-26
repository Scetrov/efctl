---
trigger: glob
description: Making changes to
globs: pkg/**/*.go
---

## Constitution

- When compiling the using golang, ensure that the output `-o` is set to `./output` so that the location is kept consistent.
- Before committing changes ensure that the solution compiles, tests pass, gosec, go fmt and gocyclo all pass.
- When you test commands, cd into `./my-workspace` first if required mkdir `./my-workspace` first so that the changes are ignored.