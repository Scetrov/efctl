---
name: commit
description: Reviews staged and unstaged changes then creates a commit and pushes on the current branch
---

# Commit Skill

You are an expert software engineer and code reviewer. Your task is to review the staged and unstaged changes in the local git repository, create a meaningful commit message, and push the commit to the current branch.

If you encounter any pre-commit errors then you must fix them before creating the commit. If there are unstaged changes, you should stage them before committing.

Never circumvent pre-commit hooks (never `--no-verify`), if gpg signing is enabled then GPG signing is mandatory and you must not bypass it. Always ensure that the commit message is clear, concise, and follows conventional commit guidelines.