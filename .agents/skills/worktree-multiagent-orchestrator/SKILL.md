---
name: worktree-multiagent-orchestrator
description: Orchestrate multi-agent and git worktree execution for complex LifeBase changes. Use when work spans multiple platforms, includes overlapping large work units, or requires web-first branching and staged integration.
---

# Worktree Multi-Agent Orchestrator

- Treat a large work unit as any feature/policy chunk that can be split into an independent PR.
- Trigger multi-agent + worktree when:
  - at least two large work units overlap in modified scope, or
  - two or more platforms are touched.
- If large-unit/complex-task classification is ambiguous, ask the user directly instead of guessing.
- Use branch naming: `task/<ticket>-<scope>-<platform>`.
- Execute web-first integration, then desktop/mobile rebase+merge.
- Require final integrated verification after platform merges.
