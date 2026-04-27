# Pinned Actions

While researching GitHub Actions for a talk, I asked myself: "How many repositories use GitHub Actions via pin-by-hash?". As I was unable to find a tool that could answer this question, I decided to build one myself.

The results are published at: [http://pin-gh-actions.kammel.dev/](http://pin-gh-actions.kammel.dev/)

## Usage

```sh
$ go run . --help
Usage of GH Pinned Actions:
  -download-dir string
        path to folder where repositories will be downloaded (default "/tmp/pinned")
  -max-pages int
        maximum number of pages to download (default 1)
  -per-page int
        number of repositories to download per page (default 100)
```

## Example

To replicate the results for 10,000 repositories, run:

```sh
go run . -max-pages 100
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
