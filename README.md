# Pinned Actions

While researching GitHub Actions for a talk, I asked myself: "How many repositories use GitHub Actions via pin-by-hash?". As I was unable to find a tool that could answer this question, I decided to build one myself.

The results are published at: [http://pin-gh-actions.kammel.dev/](http://pin-gh-actions.kammel.dev/)

## Usage

```sh
$ go run .
Usage: pinned-actions <command> [flags]

Commands:
  scan           Download repositories and run analyzers
  export-pinned  Add the pinned-action counts of a scan to the frontend's pinned.json
  export-zizmor  Export zizmor.jsonl to a slimmed per-rule-count format for the frontend
```

### scan

Downloads the most popular repositories (by stars) and runs the selected
analyzers on each of them. Results are written to `--result-dir`, one file per
analyzer (`pinned.jsonl`, `zizmor.jsonl`, `lockfile.jsonl`, `immutable.jsonl`), plus `stars.json`. Each
repository result is written as soon as it is ready, so an interrupted scan keeps
its progress; add `-resume` to continue it (see below).

```sh
$ go run . scan -h
  -analyzer string
        comma-separated list of analyzers to run (available: pinned, zizmor, lockfile, immutable) (default "pinned")
  -download-dir string
        path to folder where repositories will be downloaded (default "/tmp/pinned")
  -max-pages int
        maximum number of pages to download (default 1)
  -per-page int
        number of repositories to download per page (default 100)
  -result-dir string
        path to folder where analysis results will be written (default "results")
  -resume
        continue a previous scan: keep results in --result-dir and only analyze repositories an analyzer has no result for
```

| Analyzer | Needs | Checks |
|---|---|---|
| `pinned` | `git` | Share of `uses:` references pinned to a commit SHA / OCI digest, plus Renovate/Dependabot config |
| `zizmor` | `zizmor` on `$PATH` | [zizmor](https://github.com/zizmorcore/zizmor) findings for `.github/` |
| `lockfile` | `gh` with the `actions-lock` extension | Whether `.github/workflows/actions.lock` ([workflow lockfile](https://docs.github.com/en/actions/concepts/security/workflow-dependency-lockfiles)) exists, and how many workflows the repository has |
| `immutable` | `GITHUB_TOKEN` | Whether the latest release is marked immutable |

Set `GITHUB_TOKEN` for any real run. Without it, search requests are heavily
rate-limited and may return incomplete results.

Without `-resume`, a scan starts its result files from scratch. With `-resume`,
existing results in `--result-dir` are kept, each analyzer skips the
repositories it already has a result for (so you can add an analyzer to an
earlier run), and repositories that every analyzer is done with are not
downloaded again. Use the same flags as the interrupted run.

### export-pinned / export-zizmor

Convert scan results into the files the website reads:

```sh
go run . export-pinned --result-dir=results --year=2026
go run . export-zizmor --result-dir=results --out=frontend/zizmor-web.js
```

`export-pinned` counts the repositories that use actions as fully pinned,
partially pinned or unpinned, plus the number of repositories without actions
and the scan date (`--scanned=YYYY-MM-DD`, default: when `pinned.jsonl` was last
written). It stores them under `--year` in `frontend/pinned.json`, keeping the
other years. The website fills the percentages and the "last updated" date in its
text from the newest year. `--raw=FILE` additionally writes all per-repository results as one JSON
array, e.g. for the downloadable data archive.

`export-zizmor` also reads `stars.json`, so run it against the same
`--result-dir` as the scan.

## Example

To replicate the results for 10,000 repositories, run:

```sh
GITHUB_TOKEN=... go run . scan -max-pages 100 -analyzer pinned,zizmor,immutable
```

> [!NOTE]
> The default download directory is `/tmp/pinned`. You can change it with the `--download-dir` flag.

> [!WARNING]
> Downloading 10,000 repositories will take a long time depending on your internet connection.

## Architecture

Notes about the chosen libraries and APIs.

### GitHub Search API

We use the public GitHub [repository search API](https://docs.github.com/en/rest/search/search?apiVersion=2022-11-28#search-repositories) to request [the most popular repositories by stars](https://github.com/search?q=stars%3A10000..500000&type=repositories&ref=advsearch&s=stars&o=desc). Although the search API support pagination, it has a limit of 100 results per page, and additionally [a limit of 1000 results per search](https://docs.github.com/en/rest/search/search?apiVersion=2022-11-28#about-search).

To get around this limitation, we modify the search query after each request, and only use the first page returned.

### git

Repositories are cloned using the native `git` binary via `os/exec`, using a
partial clone combined with sparse checkout:

```sh
git clone --depth 1 --filter=blob:none --sparse <url> <dir>
git -C <dir> sparse-checkout set .github
```

`--filter=blob:none` instructs the server not to send any file blobs up front;
`sparse-checkout set .github` then fetches only the blobs under `.github/`.
Git's cone mode always includes root-level files (e.g. `README.md`, `go.mod`),
but those are negligible in size. The large savings come from skipping source
subdirectories (`src/`, `lib/`, `packages/`, etc.) that are irrelevant to the
scan. In practice this reduces disk usage from ~1.5 TB to well under 1 GB for
10,000 repositories.

### Parsing Actions

[stacklok/frizbee](https://github.com/stacklok/frizbee/tree/main/pkg/ghactions) already provides all the necessary tools to parse GitHub Actions. We use this library to parse the actions from the repositories.
