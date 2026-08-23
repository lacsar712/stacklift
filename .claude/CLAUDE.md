# Bugfix session constraints

These rules apply to this workspace. They are **not** part of the user task description.

- Analyze only from the phenomenon description and production source code.
- Do not use git history (`git show`, `git log`, `git diff`, prior commits) to find fixes.
- Do not treat test file comments, test function names, or `go test -run` as a shortcut to locate the bug.
- You may write temporary repro code to observe behavior, but form a code-path hypothesis from the phenomenon first.
- Only modify production code; do not edit `*_test.go`.
- Do not `git commit` or `git push`.
