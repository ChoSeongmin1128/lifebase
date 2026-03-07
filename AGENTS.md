# LifeBase Agent Operating Rules

## Scope
- This file applies to the entire repository.

## Mandatory Rule Loading
- Before any implementation or review, load `.claude/rules/workflow.md`.
- Load additional domain rules from `.claude/rules/*.md` based on task scope.
- Minimum mapping:
- auth/data access/sharing/sync/logging/file handling -> `.claude/rules/security.md`
- API endpoint/use case/adapter/layer boundary -> `.claude/rules/api-architecture.md`
- migration/index/rollback/schema change -> `.claude/rules/database.md`
- backend test/TDD/coverage -> `.claude/rules/testing.md`
- route state/URL/deep-link/navigation change -> `.claude/rules/routing.md`
- commit policy -> `.claude/rules/commit-message.md`
- release/version bump/sync -> `.claude/rules/versioning.md`

## Platform Execution Policy
- For every feature task, evaluate impact for Web (including responsive mobile web), Desktop (macOS/Windows), and Mobile (iOS/Android).
- If shared scope exists, implement and validate Web first.
- After Web baseline is stable, execute Desktop/Mobile in parallel and run integration validation.

## Multi-Agent / Worktree Policy
- Follow `.claude/rules/workflow.md` trigger conditions for multi-agent and worktree usage.
- If large-unit or complex-task classification is ambiguous, do not guess. Ask the user directly before proceeding.

## Codex Exec Policy
- Command execution policy is managed in `.codex/rules/default.rules`.
- Validate rule behavior with:
- `codex execpolicy check --pretty --rules ./.codex/rules/default.rules -- <command>`
- Restart Codex after adding or changing `.codex/rules/*.rules`.
