# Bug Fix Prompt Template

## Issue Information
- **Issue Number:** {{issue_number}}
- **Title:** {{issue_title}}
- **Description:**
{{issue_body}}

## Task: Fix the Bug

Please analyze this bug report and implement a fix. Follow these steps:

### 1. Understand the Bug
- Read the issue description carefully
- Identify any error messages or stack traces
- Determine the expected vs actual behavior

### 2. Explore the Codebase
- Find the relevant code files
- Understand the current implementation
- Identify the root cause

### 3. Implement the Fix
- Make minimal changes to fix the bug
- Follow the project's coding conventions
- Keep the fix focused and targeted

### 4. Test the Fix
- Verify the fix works correctly
- Run existing tests to ensure no regressions
- Add new tests if needed

## Project Context
Refer to CLAUDE.md in the project root for:
- Architecture patterns
- Code style guidelines
- Testing requirements
- API conventions

## Important Guidelines
- Don't over-engineer the solution
- Keep changes minimal and focused
- Don't add unnecessary documentation
- Don't create configuration changes unless required
- Follow existing patterns in the codebase

## Output
When complete, provide:
1. Files changed
2. Brief explanation of the fix
3. Test results
