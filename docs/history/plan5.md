# Implementation Plan: Testing Subagent for Pacman Pod Killer

**Issue:** [#13 - Implement the new testing subagent to verify the implemented feature or bugfix](https://github.com/valdisd96/pacman-pod-killer/issues/13)

**Date:** 2026-02-02

---

## Overview

Create an opencode testing subagent that automatically verifies code changes by running tests, generating missing tests, and suggesting fixes when failures occur.

### Requirements (from user)
- **Test Types:** Run existing Go unit tests, generate new tests for new features, integration/smoke tests
- **Trigger Timing:** Automatically after any code edit
- **Failure Handling:** Suggest fixes when tests fail
- **Test Coverage:** Auto-generate test stubs for all public functions

---

## Phase 1: Core Testing Subagent Configuration

**Files to Create/Modify:**
1. `.opencode/agents/tester.md` - New subagent definition
2. `.opencode/opencode.json` - Update to register the subagent

**Key Features:**
- **Multi-layer testing strategy:**
  - Layer 1: Unit test execution (`go test ./...`)
  - Layer 2: Test coverage analysis (identify untested public functions)
  - Layer 3: Integration smoke test (verify game can start without Docker)
  - Layer 4: Lint/static analysis (`go vet`, `gofmt` check)

- **Smart triggering:**
  - Run on any `.go` file modification
  - Skip if only `_test.go` files changed (avoid infinite loops)
  - Skip if only documentation changed

- **Failure response:**
  - Parse test output to identify failing tests
  - Use AI analysis to suggest specific fixes
  - Provide diff-style suggestions for simple fixes

---

## Phase 2: Test Generation Capability

**Auto-Test Generation Rules:**
1. **Detect untested public functions:**
   - Parse AST to find exported functions/methods
   - Cross-reference with existing `_test.go` files
   - Identify gaps in test coverage

2. **Generate test templates:**
   - Basic happy-path tests
   - Error case tests (where functions return errors)
   - Table-driven tests for functions with multiple inputs
   - Mock generation for Docker-dependent tests

3. **Test file organization:**
   - Follow existing patterns (see `internal/ai/sarsa_test.go`)
   - Use `package <name>` (not `package <name>_test`) for internal access
   - Place tests in same package as source code

---

## Phase 3: Integration & Smoke Tests

**Smoke Test Suite:**
- **Build test:** `go build ./...` must succeed
- **Quick start test:** Initialize game state without Docker (mock mode)
- **Maze generation test:** Generate 10 mazes with different seeds, verify connectivity
- **AI test:** Run 100 SARSA iterations without errors
- **Lint check:** `go vet ./...` + `gofmt` formatting check

---

## Phase 4: Fix Suggestion System

**Intelligent Failure Analysis:**
1. **Compilation errors:** Suggest imports, type fixes, syntax corrections
2. **Test failures:** Analyze assertion failures, suggest edge case handling
3. **Lint issues:** Auto-format with `gofmt`, suggest style fixes
4. **Race conditions:** Detect with `go test -race`, suggest synchronization fixes

---

## Implementation Details

### Subagent Definition Structure

```yaml
---
description: Automated testing and verification for Go code changes
mode: subagent
temperature: 0.1
tools:
  bash: true
  edit: true
  read: true
  write: true
  glob: true
  grep: true
permission:
  edit:
    "*_test.go": allow
    "internal/**/*.go": deny
  bash:
    "go test*": allow
    "go build*": allow
    "go vet*": allow
    "gofmt*": allow
    "git*": allow
---
```

### Testing Workflow

1. **Pre-check:** Verify Go environment, check formatting
2. **Build:** `go build ./...` - must pass
3. **Unit tests:** `go test ./...` - run all tests
4. **Coverage analysis:** Parse coverage report, find untested functions
5. **Test generation:** Create missing tests (if enabled)
6. **Smoke tests:** Run integration tests
7. **Report:** Summary of results with fix suggestions

---

## File Structure

```
.opencode/
├── opencode.json          # Add tester agent config
└── agents/
    ├── tester.md          # NEW: Testing subagent definition
    ├── github.md
    └── docs-writer.md
```

---

## Testing Packages Priority

Based on the codebase structure, test generation priority:
1. `internal/ai/` - Already has tests, ensure full coverage
2. `internal/game/` - Core game logic, needs comprehensive tests
3. `internal/maze/` - Maze generation algorithms
4. `internal/dockerwatch/` - Docker integration (mock-based tests)
5. `internal/render/` - Rendering logic
6. `internal/input/` - Input handling
7. `internal/logger/` - Logging system

---

## Success Criteria

- [ ] Tester agent runs automatically on Go file changes
- [ ] All existing tests pass (`go test ./...`)
- [ ] Missing tests are generated for untested public functions
- [ ] Smoke tests verify game can start
- [ ] Failed tests trigger fix suggestions
- [ ] No infinite loops (test generation doesn't trigger more tests)
- [ ] Respects opencode permission model

---

## Rollout Strategy

1. **Phase 1:** Create tester subagent with basic test execution
2. **Phase 2:** Add test coverage analysis
3. **Phase 3:** Implement auto-test generation
4. **Phase 4:** Add smoke tests and fix suggestions
5. **Phase 5:** Full integration with automatic triggers

---

## Notes

- Follow existing AGENTS.md guidelines for Go code style
- Use table-driven tests as shown in `internal/ai/sarsa_test.go`
- Avoid Docker-dependent tests in automated testing (use mocks)
- Ensure tests are deterministic (use seeds where applicable)
- Respect the destructive nature of the game (container deletion)
