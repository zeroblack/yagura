# Security Policy

## Supported versions

yagura is pre-1.0 and ships from a single line of development. Security fixes target the
**latest released version**; please upgrade before reporting.

## Reporting a vulnerability

**Do not open a public issue for security problems.**

Report privately through GitHub's [private vulnerability
reporting](https://github.com/zeroblack/yagura/security/advisories/new) (the "Report a
vulnerability" button on the repository's *Security* tab). Include:

- a description of the issue and its impact,
- the version affected (`yagura --version`),
- the smallest steps to reproduce it.

You can expect an initial response within a few days. Once a fix is ready, a patched
release is published and the advisory is disclosed with credit to the reporter, unless you
prefer to stay anonymous.

## Scope notes

yagura is **read-only**: it never runs a git command that writes and never executes code
from the repositories it observes. It does read local files — git metadata, and AI agent
session transcripts under the configured `agents.claude_root`. Reports that involve
yagura mishandling those inputs (path traversal, resource exhaustion, leaking transcript
contents) are in scope.
