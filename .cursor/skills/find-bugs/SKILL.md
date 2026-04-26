---
name: find-bugs
description: >-
  Inspects recent commits and code changes for high-severity correctness bugs
  (data loss, crashes, security issues, significant user-facing breakage). Use
  when the user asks for bug hunting, a deep review of recent changes, critical
  defect analysis, or pre-merge correctness checks focused on serious issues only.
---

# Find critical bugs

## When to use

Apply when the user wants a **narrow, high-severity** pass over recent work: correctness and safety, not style or nitpicks. Default expectation is often **no critical bugs found**; that is a successful outcome.

## Investigation strategy

1. **Scope** — Prefer `git log`, `git diff`, and tracing from the changed surface area. Focus on behavioral changes with meaningful blast radius.

2. **Look for** — Data corruption; races that lose writes; null dereferences on critical paths; auth or permission bypasses; infinite loops; resource leaks; silent truncation or lossy transforms.

3. **Trace fully** — Follow caller chains and downstream effects. Do not stop at pattern-matching the diff.

4. **Ignore** — Style; minor edge cases; purely theoretical issues without a concrete trigger; low-severity UX degradation.

## Confidence bar

- Only treat something as a **confirmed bug** if you can describe a **concrete scenario** that triggers it.
- If you cannot construct a plausible trigger, do **not** present it as a fix-ready defect; note it as uncertain or omit it.
- When impact is real but fix confidence is low, report findings in the conversation **without** shipping a speculative code change as if it were certain.

## Fix strategy (only if a critical bug is validated)

- Implement a **minimal**, high-confidence fix.
- Add or update tests when practical to lock behavior.
- Avoid broad refactors in the same change set.

## Safety rules

- Do **not** treat uncertain issues as mandatory fixes.
- If no critical bug is found, give a short **“No critical bugs found”** summary and what was reviewed.

## Output template

When reporting a validated issue (and optional fix), use:

```markdown
## Bug and impact

[What breaks, for whom, and how bad.]

## Root cause

[Why the code behaves wrongly — linkage to the faulty assumption or missing guard.]

## Fix and validation

[What changed; tests run or scenarios verified.]
```

For a clean review with nothing critical:

```markdown
## Summary

No critical bugs found.

**Reviewed:** [scope — e.g. commits, files, or feature area]
```
