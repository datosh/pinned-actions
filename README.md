# Pinned Actions

## Goal

Use the public [list repositories API](https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#list-public-repositories) to enumerate all repositories and check whether they use GitHub actions via pin-by-hash.

Instead of doing an exhaustive search, we should start with the most popular (star count) repositories. We can use the [repository search API](https://docs.github.com/en/rest/search/search?apiVersion=2022-11-28#search-repositories) for this, e.g., [https://github.com/search?q=stars%3A%3E1000&type=Repositories&ref=advsearch&l=&l=&s=stars&o=desc](https://github.com/search?q=stars%3A%3E1000&type=Repositories&ref=advsearch&l=&l=&s=stars&o=desc).

## Estimations

As of June 2023, GitHub is estimated to host [at least 28 million public repositories](https://en.wikipedia.org/wiki/GitHub).

GitHub exposes information via it's public REST API, which has a rate limit of [60 requests / hour for unauthenticated users](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#primary-rate-limit-for-unauthenticated-users) and [5000 requests / hour for authenticated users](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#primary-rate-limit-for-authenticated-users).

Assuming we only need a single API request per repository:

5.000 requests / hour * 24 = 120.000 requests / day.
28.000.000 public repositories / 120.000 requests / day = 233 days.

This would take us the better part of a year to enumerate all public repositories.

## Resources

Can we use [stacklok/frizbee](https://github.com/stacklok/frizbee/tree/main/pkg/ghactions) to reuse parsing of actions file?
