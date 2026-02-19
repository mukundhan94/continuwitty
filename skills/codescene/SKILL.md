---
name: codescene
description: Analyze and improve code health using CodeScene MCP tools. Use when the user mentions code health, code smells, technical debt, refactoring, or CodeScene.
user-invocable: true
argument-hint: [file-or-action]
---

# CodeScene Code Health

Analyze code health, fix issues, and validate changes using CodeScene's MCP tools.

## Usage

```
/codescene                          # Run pre-commit safeguard on current changes
/codescene review path/to/file      # Deep review a specific file
/codescene hotspots                 # List technical debt hotspots
/codescene fix path/to/file         # Review, fix, and test a file
/codescene goals                    # Show technical debt goals
```

## Workflow

Follow these steps based on the action requested:

### Default (no args or `check`): Pre-commit safeguard

1. Call `pre_commit_code_health_safeguard` to validate uncommitted changes
2. If code health regressed, report the issues clearly
3. Suggest specific fixes for each regression

### `review <file>`: Deep code health review

1. Call `code_health_review` on the specified file
2. Report the code health score and all identified issues
3. List each issue with its location and severity
4. Suggest concrete fixes ordered by impact

### `hotspots`: Technical debt hotspots

1. Call `list_technical_debt_hotspots` to find high-risk areas
2. Present results as a prioritized table: file, score, change frequency
3. Recommend which files to address first (worst score + most changed)

### `goals`: Technical debt goals

1. Call `list_technical_debt_goals`
2. Present active goals and progress

### `fix <file>`: Review, fix, and test

This is the main workflow. Follow these steps strictly:

1. **Review** - Call `code_health_review` on the target file. Note the starting score.
2. **Read** - Read the file to understand the code.
3. **Plan fixes** - From the review results, pick the highest-impact issues to fix. Focus on:
   - Complex conditionals (simplify or extract)
   - Long functions (break into smaller ones)
   - Deep nesting (flatten with early returns)
   - Code duplication (extract shared logic)
   - Naming issues (rename for clarity)
4. **Apply fixes** - Edit the file. Make small, focused changes. Do NOT change behavior.
5. **Validate** - Call `pre_commit_code_health_safeguard` to confirm the score improved.
6. **Test** - Run the project's test suite to ensure nothing broke:
   - For this project: `make test`
   - If tests fail, revert the breaking change and try a different approach.
7. **Report** - Show before/after scores and what changed.

If the score did not improve after step 5, try a different refactoring approach. Repeat up to 3 times.

## Rules

- Never change observable behavior. Refactoring only.
- Make small, incremental edits. Re-validate after every 3-5 changes.
- Target code health score of 10.0. Anything below 9 needs attention.
- If `code_health_auto_refactor` is available (ACE), prefer it for automated fixes.
- Always run tests after making changes.
- If a file's score is already 10.0, say so and move on.
