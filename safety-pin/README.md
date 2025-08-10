# Project: Safety Pin

Given that there was little movement in the last year of repositories pinning
their dependencies (~2% -> ~3%), this project will aim to accelerate the
process by proactively engaging with repositories to pin their dependencies.

To maximize the likelihood of success, we will first approach repositories that:
+ are partially pinned, i.e., we know they are not opposed to pinning their
  dependencies
+ have a renovate or dependabot configuration, i.e., already have a process in
  place to update their dependencies, so the argument of using `main` or `v1` to
  receive updates is not as likely

Starting out, we will use a manual approach to learn about the process and usual
feedback received from maintainers. Later, we might use a more automated approach,
e.g., create issues / raise PRs, or even develop a GitHub App to help with the
process.

## Process

To be conservative, we will start by creating an issue in the identified repository
to gauge whether the maintainer is open to the idea of pinning their dependencies.

If the maintainer is open to the idea, we will then create a PR to pin the
dependencies.

If the maintainer is not open to the idea, we will then close the issue.

### Issue Template

I've noticed that this project is pinning most of its GitHub Action dependencies by referencing a commit hash. This is great, as it ensures that the workflows are both stable and secure. It is a security best practice, [endorsed by GitHub](https://docs.github.com/en/actions/how-tos/security-for-github-actions/security-guides/security-hardening-for-github-actions#using-third-party-actions), and helps prevent security incidents such as [CVE-2025-30066](https://www.wiz.io/blog/github-action-tj-actions-changed-files-supply-chain-attack-cve-2025-30066), aka the "tj-actions/changed-files supply chain attack".

I'd like to know if you're open to the idea of pinning all GitHub Action dependencies.

If so, I'll be happy to create a PR to pin the remaining dependencies.

Thanks!

### PR Template

I've noticed that this project is pinning most of its GitHub Action dependencies by referencing a commit hash. This is great, as it ensures that the workflows are both stable and secure. It is a security best practice, [endorsed by GitHub](https://docs.github.com/en/actions/how-tos/security-for-github-actions/security-guides/security-hardening-for-github-actions#using-third-party-actions), and helps prevent security incidents such as [CVE-2025-30066](https://www.wiz.io/blog/github-action-tj-actions-changed-files-supply-chain-attack-cve-2025-30066), aka the "tj-actions/changed-files supply chain attack".

I was not able to find any legitimate reason in the git history or issues, so I assume these were just overlooked.

This PR pins the actions to match the best practice followed by the other workflows in this repository.

### Implementation

1. List actions with frizbee
  ```sh
  frizbee actions list
  ```
1. Migrate to pinned actions
  ```sh
  frizbee actions
  ```

## Statistics

### Potential Repositories

+ grafana/grafana
+ grafana/loki (only "official" actions/)
+ hcavarsan/kftray
+ hashintel/hash

### Merged PRs

+ https://github.com/cloudnative-pg/cloudnative-pg/pull/8023
+ https://github.com/cilium/tetragon/pull/3986
+ https://github.com/element-hq/element-web/pull/30501

### Open PRs

+ https://github.com/hashicorp/terraform-provider-kubernetes/pull/2765
+ https://github.com/trueforge-org/truecharts/pull/38247
+ https://github.com/DefectDojo/django-DefectDojo/pull/12958

## Feedback

We will collect the project's feedback here to improve the process.

## TODO

Explore if we can use CodeQL to identify repositories that are partially pinned.
+ https://github.com/github/codeql/blob/main/actions/ql/src/Security/CWE-829/UnpinnedActionsTag.ql

## Resources

Grafana Labs team has
[written about](https://grafana.com/blog/2025/06/26/how-to-detect-vulnerable-github-actions-at-scale-with-zizmor/)
using Zizmor with GitHub Actions at scale. They are using a
[central action](https://github.com/grafana/security-github-actions/blob/main/.github/workflows/self-zizmor.yaml)
to scan all repositories in their organization.
