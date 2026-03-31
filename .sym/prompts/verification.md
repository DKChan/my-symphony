# Verification Prompt

You are in the verification phase of a software development workflow. Your task is to generate a comprehensive verification report based on the completed implementation.

## Context

You have completed the implementation phase. Now you need to:
1. Run the test suite and collect results
2. Validate against BDD scenarios
3. Generate a verification report

## Instructions

### Step 1: Run Tests
Execute the project's test suite:
- Run unit tests
- Run integration tests if available
- Collect test results (pass/fail counts)

### Step 2: Validate BDD Scenarios
For each BDD scenario in the approved BDD rules:
- Check if the implementation satisfies the Given-When-Then conditions
- Document pass/fail status for each scenario

### Step 3: Generate Verification Report
Create a verification report at `docs/verification_report.md` with the following structure:

```markdown
# Verification Report

**Task ID:** {task_identifier}
**Task Title:** {task_title}
**Generated At:** {timestamp}

## Test Results Summary

| Metric | Count |
|--------|-------|
| Total Tests | {total} |
| Passed | {passed} |
| Failed | {failed} |
| Skipped | {skipped} |

## Test Details

### Passed Tests
{list of passed tests}

### Failed Tests
{list of failed tests with error messages}

## BDD Scenario Validation

| Scenario | Status | Notes |
|----------|--------|-------|
| {scenario_name} | {pass/fail} | {notes} |

## Implementation Summary

{brief summary of what was implemented}

## Recommendations

{any recommendations for improvement or follow-up}

## Conclusion

- [ ] All tests passed
- [ ] All BDD scenarios validated
- [ ] Implementation meets requirements

**Overall Status:** {PASS/FAIL}
```

## Important Notes

- Be thorough in testing - don't skip edge cases
- Document all failures with enough detail for debugging
- If tests fail, still generate the report with failure details
- The report should be clear and actionable for reviewers

## Output

After generating the report:
1. Save it to `docs/verification_report.md`
2. Print a summary of the verification results
3. Indicate whether the implementation is ready for acceptance review