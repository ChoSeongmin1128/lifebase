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

## Documentation Sync Policy
- If code, config, routes, schema, platform support, or user-visible behavior changes, update the related docs in the same task.
- Minimum sync targets are `README.md`, `docs/700-마일스톤.md`, `plan.md`, and affected rule files under `.claude/` or `.codex/`.
- If the touched scope already has known documentation drift, fix that drift before considering the task complete.

## Platform Execution Policy
- For every feature task, evaluate impact for Web (including responsive mobile web), Desktop (macOS/Windows), and Mobile (iOS/Android).
- If shared scope exists, implement and validate Web first.
- After Web baseline is stable, execute Desktop/Mobile in parallel and run integration validation.

## Multi-Agent / Worktree Policy
- Default working context is the current repository's `dev` branch.
- Do not create or switch to a git worktree unless the user explicitly asks for it.
- Follow `.claude/rules/workflow.md` for multi-agent usage and for worktree handling after an explicit user request.
- If large-unit or complex-task classification is ambiguous, do not guess. Ask the user directly before proceeding.

## Codex Exec Policy
- Command execution policy is managed in `.codex/rules/default.rules`.
- Validate rule behavior with:
- `codex execpolicy check --pretty --rules ./.codex/rules/default.rules -- <command>`
- Restart Codex after adding or changing `.codex/rules/*.rules`.
