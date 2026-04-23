---
name: create-pr
description: >-
  Creates a feature branch, commits and pushes local changes, and produces a
  pull-request title and body from the actual diff. Use when the user wants to
  open a PR, prepare a pull request, push a branch, or get PR description text
  from their changes.
---

# Create pull request

## When to use

Apply this skill when the user intent is: ship work via a PR, get review-ready text, or sync a branch to origin before opening the PR in the host (GitHub, GitLab, etc.).

## Workflow

1. **Inspect the repo** — Run `git status` and read recent commits. Use `git diff` for unstaged and `git diff --cached` for staged changes; include `git log -1` / short range if needed for context. Prefer the real diff over guessing.

2. **Branch** — If the user is on the default branch (e.g. `main` / `master`) with new work, create a feature branch with a short, kebab-style name (e.g. `docs/constitution-v1` or `feat/daily-recommendation-copy`). If already on a non-default branch, keep it unless the user asked to rename.

3. **Commit** — Stage and commit with a message that matches the change (imperative mood, Conventional Commits if the project uses them: `type(scope): summary`). If there is nothing to commit, report that and do not create an empty commit.

4. **Push** — `git push -u origin <branch>` when a remote named `origin` exists. If push fails, diagnose (auth, no upstream, protected branch) and report clearly.

5. **Open the PR in the host** — The agent cannot click the “Open PR” button. After push, give the user the PR creation URL if inferable (e.g. `https://github.com/<org>/<repo>/compare/<branch>?expand=1`) or the compare path; otherwise state they should open a PR from the pushed branch in their Git UI.

6. **Deliverables** — Always output: (A) a **PR title**, (B) a full **PR body** in Markdown per [PR body template](#pr-body-template), (C) the branch name and push status.

## PR body template

Use this structure. Omit sections that do not apply; keep **Breaking changes**, **Performance**, **Security**, and **Testing** with “None” or “N/A” only when true.

```markdown
## Description

[Why the change was needed — problem, goal, or ticket.]

[What changed in plain language—behavior, not file-by-file unless small.]

## Major changes

- [User-visible or architecturally significant item]
- […]

## Breaking changes

[None, or list with migration / caller impact.]

## Performance

[None, or what was measured, expected impact, and risk.]

## Security

[None, or data handling, auth, secrets, trust boundaries.]

## Testing

[How this was tested: commands run, manual checks, or “not run” with reason.]
```

### Title

- One line, imperative, **≤72 characters** when possible.
- Summarizes the **main** outcome, not every file (e.g. `docs: ratify constitution v1.0.0 and sync plan gates`).

## Quality bar

- **Ground the description in the diff** — Do not invent features, paths, or risks not supported by the changes.
- **Call out** migrations, config/env changes, and anything operators must do after deploy.
- If the diff mixes unrelated concerns, suggest **splitting into separate PRs** in the body.

## Edge cases

| Situation | Action |
|----------|--------|
| No remote / cannot push | Produce title + body anyway; say push failed and what to fix. |
| User only wants text (no push) | Generate title + body from `git diff` / `git diff main...HEAD` without pushing. |
| Massive diff | Lead with a short summary, then “Major changes” bullets; link to “large refactor” and key entry points. |

## Optional: project rules

If the repository has a `PULL_REQUEST_TEMPLATE` or `CONTRIBUTING.md`, align section headings and required fields with that file when read.
