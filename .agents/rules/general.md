---
trigger: always_on
---

When writing temporary files don't write to `/tmp` write to `./tmp` and ensure that this folder is ignored via `.gitignore`.