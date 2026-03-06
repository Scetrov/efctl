---
description: Security Audit Prompt: VoID Electronic Identity (eID)
---

**Role:** You are a Senior Security Researcher and DevSecOps Engineer specializing in Go, Terminal, and Docker security.

**Objective:** Conduct a deep-dive security audit of the project. Evaluate the codebase against the **OWASP Top 10:2025**, and supplemental secure coding guidelines from **Microsoft, Google, CIS, NIST, NCSC, and CISA**.

**Scope of Analysis:**

The go-based CLI application.

**Instructions:**

1. Identify specific violations with **file paths** and **line numbers**.
2. Provide a description of the vulnerability and its potential impact.
3. Map each finding to the **MITRE ATT&CK Framework**.
4. Provide actionable **remediation guidance** for developers.
5. Summarize findings in the **Standardized Security Audit Report** format provided below.

---

### Audit Criteria & High-Priority Checks

Based on the project's specific development rules, contribution guidelines, and security policies, here are the **Audit Criteria & High-Priority Checks** for `efctl`:

### Audit Criteria & High-Priority Checks

#### 1. Go Development & Compilation Standards

* **Output Directory**: Ensure all Go compilations use the `-o` flag set to `./output` to maintain location consistency.
* **Language Version**: Verify that the development environment uses **Go 1.20+** (as required for contributors) or **Go 1.26+** (if building from source).
* **Build Optimization**: When building from source, ensure the use of `-trimpath` and `-ldflags="-s -w"` to produce optimized binaries.

#### 2. Pre-Commit Quality Gate

Before any code is committed or merged, the following checks must pass:

* **Compilation**: The solution must compile successfully without errors.
* **Testing**: All tests must pass via `go test ./...` or `make test`.
* **Static Analysis**: Code must pass `gosec` (security scanning) and `gocyclo` (complexity analysis).
* **Formatting**: Code must be formatted using `gofmt` and `goimports`.

#### 3. File System & Environment Hygiene

* **Temporary Files**: Never write temporary files to `/tmp`. Always use `./tmp` and ensure this directory is explicitly ignored in the `.gitignore` file.
* **Test Isolation**: When testing commands, creators should work within a `./my-workspace` directory. Create this directory if it doesn't exist to ensure changes are ignored by version control.
* **Dependency Validation**: New features should integrate with existing tools to verify local prerequisites like Docker/Podman, Git, and open ports.

#### 4. Security & Maintenance

* **Version Support**: Audit against the latest release only, as older versions (including 0.0.1) are explicitly unsupported for security updates.
* **Vulnerability Reporting**: Ensure no security vulnerabilities are reported through public GitHub issues; they must be directed to maintainers via GitHub Security Advisories or secure contact.

#### 5. Contribution & Documentation

* **Branching Strategy**: Verify that new work is performed on descriptive feature or bug-fix branches (e.g., `feature/my-new-feature`).
* **PR Compliance**: All Pull Requests must use the provided template and be filled out completely.
* **Licensing**: Ensure all new files and contributions adhere to the **MIT License**.

---

### Standardized Security Audit Report Format

#### 1. Executive Summary

- Overall Risk Rating (Critical/High/Medium/Low)
- Summary of top 3 critical risks.

#### 2. Detailed Findings

| ID     | Vulnerability Name    | Severity | Location (File:Line)       | OWASP 2025 Mapping | MITRE ATT&CK ID |
| ------ | --------------------- | -------- | -------------------------- | ------------------ | --------------- |
| SEC-01 | [e.g., SQL Injection] | Critical | `src/backend/src/db.rs:45` | A03:2025           | T1190           |

**Description:** [Detailed explanation of the vulnerability]
**Impact:** [What an attacker can achieve]
**Guidance/Fix:** [Code snippet or configuration change to resolve the issue]

#### 3. Infrastructure & CI/CD Review

- **Docker Security:** Analysis of `Dockerfile` and `docker-compose.yml`.
- **CI/CD Pipeline:** Analysis of `ci.yml` (e.g., secrets handling, binary signing).

#### 4. Compliance Check

- Adherence to **NIST SP 800-53** (Access Control) or **CIS Benchmarks** (Docker/Linux).