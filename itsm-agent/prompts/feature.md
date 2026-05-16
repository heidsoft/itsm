# Feature Implementation Prompt Template

## Issue Information
- **Issue Number:** {{issue_number}}
- **Title:** {{issue_title}}
- **Description:**
{{issue_body}}

## Task: Implement Feature

Please analyze this feature request and implement it. Follow these steps:

### 1. Understand Requirements
- Read the feature description carefully
- Clarify any ambiguities
- Identify the scope of the feature

### 2. Explore the Codebase
- Find related components
- Understand existing patterns
- Identify what needs to be extended

### 3. Plan the Implementation
- Determine the components to modify
- Design the solution following project conventions
- Consider edge cases

### 4. Implement the Feature
- Add new functionality
- Modify existing code as needed
- Follow coding conventions
- Add appropriate tests

### 5. Verify Implementation
- Test the new feature works
- Ensure existing tests pass
- Check for any edge cases

## Project Context
Refer to CLAUDE.md in the project root for:
- Architecture patterns
- API conventions
- Frontend component patterns
- Backend service patterns

## Important Guidelines
- Don't over-engineer the solution
- Keep changes minimal and focused
- Follow existing patterns
- Don't add unnecessary features
- Don't create documentation unless explicitly requested

## Output
When complete, provide:
1. Files changed
2. Brief explanation of the implementation
3. Test results
