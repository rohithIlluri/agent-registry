---
name: git-workflow
description: "Opinionated Git workflow: conventional commits, branch naming, PR descriptions with test plans, changelog generation."
version: "1.0.3"
license: MIT
keywords: [git, commits, branching, changelog, pr, conventional-commits]
category: devops
allowed-tools: [Bash, Read]
user-invocable: true
---

# Git Workflow

Enforce a clean, conventional Git workflow across commit messages, branch names, PR descriptions, and changelogs.

## When to invoke

Invoke when the user asks to commit, create a PR, name a branch, or generate a changelog.

## Conventional Commits

Commit messages must follow: `<type>(<scope>): <description>`

Types: `feat` | `fix` | `docs` | `style` | `refactor` | `perf` | `test` | `build` | `ci` | `chore` | `revert`

Rules:
- Subject line ≤ 72 characters, imperative mood ("add" not "added")
- Body explains WHY, not what (the diff shows what)
- Breaking changes: add `BREAKING CHANGE:` footer or `!` after type
- Reference issues with `Closes #123` or `Fixes #456` in footer

## Branch naming

`<type>/<short-description>` — e.g., `feat/user-auth`, `fix/null-pointer`, `chore/upgrade-deps`

## PR description template

```markdown
## Summary
- <bullet 1>
- <bullet 2>

## Test plan
- [ ] Unit tests pass
- [ ] Manual smoke test: <steps>
- [ ] Edge cases: <list>
```

## Changelog generation

When asked to generate a changelog:
1. Run `git log --oneline <previous-tag>..HEAD`
2. Group commits by type
3. Output as `## [version] — YYYY-MM-DD` with `### Features`, `### Bug Fixes`, `### Breaking Changes` sections

## Constraints

- Never amend published commits; create new ones instead.
- Never force-push to main/master.
- Always show the full commit message before committing.
