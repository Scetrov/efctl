# Agent Constitution

This document defines the core principles and operational constraints for any AI Agent working on the `efctl` repository.

## 1. Test-First Development

All features and bug fixes must follow a test-driven approach. Create or update tests before or alongside implementation to ensure the product remains fully verified.

## 2. Testing Pyramid Adherence

Maintain a healthy testing balance according to the pyramid:

- **Unit Tests:** High volume, testing individual functions and logic in isolation.
- **Integration Tests:** Medium volume, testing interactions between modules (e.g., container operations, CLI commands).
- **E2E Tests:** Low volume, verifying full user flows across the entire environment.

## 3. Clean Code & Quality Gates

- Write clean, maintainable, and idiomatic Go code.
- Always run `pre-commit` hooks locally to ensure all linting, formatting, and security gates pass before finalizing a task.

## 4. Security-First

Prioritize security in every change. Avoid hardcoding credentials, ensure proper file permissions (e.g., `0600` for sensitive logs), and utilize security scanning tools like `gosec`.

## 5. Independent Operation

The Agent is authorized to operate independently. Continue working through sub-tasks and error resolution unattended until the primary objective is fully complete or a critical blocker is reached.

## 6. Environmental Isolation

To maintain a non-breaking flow and avoid permission requests:

- All temporary files, intermediate build artifacts, or scratchpads must be stored in the `./tmp` directory.
- Do not write to system-level directories or paths outside the repository root.
