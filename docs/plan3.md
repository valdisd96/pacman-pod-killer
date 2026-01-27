# Documentation Subagent Implementation Plan

Issue: https://github.com/valdisd96/pacman-pod-killer/issues/2

## Overview

Create an OpenCode subagent that automatically updates project documentation when code changes occur. This follows AI coding agent best practices from the AGENTS.md standard and OpenCode's native subagent system.

## Best Practices Research Summary

| Practice | Description |
|----------|-------------|
| **Single Responsibility** | Subagent handles ONLY documentation updates, not code changes |
| **Read-First, Write-Last** | Read code to understand changes, then write only to doc files |
| **Scoped Tool Permissions** | Limit tools to read + edit; disable bash, webfetch, websearch |
| **Low Temperature** | Use 0.1-0.2 for deterministic, consistent documentation |
| **Human-in-the-Loop** | Review doc changes before commit (use `permission.edit: ask`) |
| **Version Control** | Keep agent config in `.opencode/agents/` and commit to git |

## Implementation

### File to Create

`.opencode/agents/docs-writer.md`

### Configuration

```yaml
---
description: Updates project documentation (README.md, AGENTS.md, docs/) when code changes
mode: subagent
temperature: 0.2
tools:
  bash: false
  webfetch: false
  websearch: false
  codesearch: false
permission:
  edit: allow
---
```

### System Prompt

```markdown
You are a technical documentation specialist for a Go terminal game project (pacman-pod-killer).

## Your Responsibilities
1. Keep README.md and AGENTS.md in sync with code changes
2. Update docs/ files when relevant features change
3. Document new flags, commands, and behaviors
4. Maintain consistent formatting and terminology

## Documentation Files to Maintain
- `README.md` - User-facing documentation (features, usage, flags)
- `AGENTS.md` - AI agent instructions (build commands, code style, architecture)
- `docs/*.md` - Technical specifications and plans

## Guidelines
- Match existing documentation style and tone
- Keep README concise for humans; keep AGENTS.md detailed for AI tools
- Update flag tables when defaults or descriptions change
- Reflect new packages in the "Repository layout" section of AGENTS.md
- Do NOT add emojis unless already present in the file
- Use code blocks with language hints (```bash, ```go)
- Keep line length ≤100 characters where practical

## Trigger Conditions
Run after changes to:
- `cmd/pacman/main.go` (new flags)
- New packages in `internal/`
- Significant behavior changes in game logic
- AI-related changes in `internal/ai/`
```

## Usage

### Manual Invocation
```
@docs-writer Please update docs to reflect the new --container-filter flag I added
```

### Automatic Invocation
The Build agent will automatically invoke docs-writer when appropriate based on the description.

### Example Workflow
1. Developer adds new `--filter` flag to `cmd/pacman/main.go`
2. Developer invokes: `@docs-writer sync documentation with new filter flag`
3. Subagent reads the code changes
4. Subagent proposes updates to README.md (flag table) and AGENTS.md (flags section)
5. Developer reviews diff and commits

## Documentation Update Matrix

| Code Change | README.md Update | AGENTS.md Update |
|-------------|------------------|------------------|
| New CLI flag | Flag table | Flags section |
| New package | - | Repository layout |
| New AI mode | AI Modes section | SARSA/AI sections |
| Build change | Build section | Build commands |
| New game mechanic | Features | Game loop behavior |

## Alternative: Comprehensive Docs Agent

For teams requiring stricter review, use `permission.edit: ask` and add:

```yaml
model: anthropic/claude-sonnet-4-20250514
maxSteps: 10
```

This forces human approval before any documentation edit.

## Implementation Steps

1. [ ] Create directory `.opencode/agents/` if not exists
2. [ ] Create `.opencode/agents/docs-writer.md` with configuration above
3. [ ] Test with: `@docs-writer describe what documentation you would update for this project`
4. [ ] Verify subagent appears in `@` autocomplete menu
5. [ ] Close issue #2

## References

- [OpenCode Agents Documentation](https://opencode.ai/docs/agents)
- [AGENTS.md Standard](https://agents.md/)
- [Claude Code Subagents Best Practices](https://docs.anthropic.com/en/docs/claude-code/sub-agents)
- [Factory AGENTS.md Guide](https://docs.factory.ai/cli/configuration/agents-md)
