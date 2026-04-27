# Multi-Analyzer Architecture Plan

## Goal

Extend the tool to support multiple, pluggable analyzers beyond the existing
pin-by-hash check. Each analyzer produces its own JSON output file. The
download phase is unchanged: repos are sparse-cloned once and reused across
runs.

## Decisions

| Topic | Decision |
|---|---|
| Analyzer selection | `--analyzer=pinned,zizmor` opt-in flag; `pinned` is the default |
| Zizmor binary | Must be on `$PATH`; tool validates at startup and exits early with a clear error |
| Persistence | JSON — one file per analyzer under `--result-dir` |
| Re-runs | Existing behavior: repo already on disk → skip clone, still analyze |
| Frontend | Unchanged; `export` subcommand translates `pinned.json` → existing web format |

## Analyzer classification

- **Offline** (work on local checkout only): `pinned`, `zizmor`
- **Online** (need GitHub API): `immutable`

Online analyzers receive a `*github.Client` via their constructor. The
`Analyzer` interface is identical for both — callers do not need to know.

## Commit sequence

### Phase 1 — Scaffolding (no behavior change; tests pass after every commit)

| # | Commit | Files | What changes |
|---|---|---|---|
| 1 | `refactor: extract Analyzer interface` | `analyzer.go` (new) | 3-method interface: `Name() string`, `Analyze(ctx, repoPath, repo) error`, `Close() error` |
| 2 | `refactor: wrap existing logic in PinnedAnalyzer` | `analysis.go` → `analysis_pinned.go` | `AnalyseRepository` becomes a method on `PinnedAnalyzer`; struct accumulates results; `Close()` writes `pinned.json`; tests updated |
| 3 | `refactor: update runner to use []Analyzer` | `main.go` | Loop calls `Analyze()` per repo, `Close()` per analyzer at end; only `PinnedAnalyzer` registered; net behavior identical |
| 4 | `feat: replace --result-file with --result-dir` | `config.go`, `args.go`, `main.go` | `--result-file` removed, `--result-dir` added (default `./results`); each analyzer writes `{dir}/{name}.json` |
| 5 | `feat: add --analyzer opt-in flag` | `config.go`, `args.go`, `main.go` | `--analyzer=pinned` (default); comma-separated; validated at startup against registered names; builds the `[]Analyzer` slice |

### Phase 2 — New analyzers

| # | Commit | Files | What changes |
|---|---|---|---|
| 6 | `feat: add ZizmorAnalyzer` | `analysis_zizmor.go` (new) | Validates `zizmor` on `$PATH` at construction (fails fast); shells out `zizmor --format=json .github/` per repo; `Close()` writes `zizmor.json` |
| 7 | `feat: add ImmutableReleasesAnalyzer` | `analysis_immutable.go` (new) | Receives `*github.Client` in constructor; calls `GET /repos/{owner}/{repo}/releases/latest`; 404 → `false`; `Close()` writes `immutable.json` |

### Phase 3 — Export subcommand

| # | Commit | Files | What changes |
|---|---|---|---|
| 8 | `feat: add export subcommand` | `export.go` (new), `main.go` | `pinned-actions export --result-dir=results/ --out=web.json`; reads `pinned.json`, maps to existing frontend JSON shape; dispatch via stdlib `flag.FlagSet` |

## Data shapes

### `results/pinned.json` — unchanged, backwards compatible
```json
[{"repository":"owner/name","actions_pinned":3,"actions_total":5,"has_renovate":true,"has_dependabot":false}]
```

### `results/zizmor.json` — one entry per repo
```json
[{"repository":"owner/name","findings":[{"rule":"template-injection","severity":"high","file":".github/workflows/ci.yml","line":12}]}]
```

### `results/immutable.json` — one entry per repo
```json
[{"repository":"owner/name","latest_release_immutable":true}]
```
