# Lockfile fixtures

`locked/` is a verbatim copy of this repository's workflows and the
`actions.lock` that `gh actions-lock` (v0.4.1) generated for them. The other
directories are derived from it by hand, as inputs for our own parsing; they
are not output of the CLI.

| Directory | Derived by | Workflows | Listed in lockfile |
|---|---|---|---|
| `locked/` | copy | `ci.yml`, `zizmor.yml` | both |
| `stale/` | `ci.yml` gained a `uses: actions/cache@v4` the lockfile lacks | 2 | both |
| `not-onboarded/` | lockfile lists only `ci.yml` | 2 | 1 |
| `malformed/` | lockfile is not valid YAML | 1 | - |

`verify-local-no-lockfile.json` is the real `gh actions-lock --verify-local
--json` output for this repository before it had a lockfile.
