# Working Rules

## Deliver a working result quickly
- Prioritize completing the task and getting it working.
- Keep changes focused. Avoid unrelated refactoring.
- Run only checks that help verify the actual change.
- Do not repeat successful checks unless relevant code changes.

## Keep tests minimal
- Add tests only when they protect important behavior or catch a meaningful regression.
- Do not write tests for every small change or trivial implementation detail.
- Prefer existing tests and focused checks over new test suites.
- Run the smallest relevant set of tests.

## Verify completion
- Before finishing, check every requirement in the user's request.
- Verify that the result works as intended using focused tests, direct inspection, or a manual check.
- Fix any issues found. Do not stop at a partial solution.
- Report what changed, what was verified, and any remaining limitations.
- Never claim success without evidence.
