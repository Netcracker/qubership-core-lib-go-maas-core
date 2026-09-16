# Changelog

All notable changes to this library are documented here.

## [Unreleased]


## [3.6.2] - 2026-09-17

### Fixed

- The resty client the MaaS clients are built with no longer retries on its own.
  Both layers retried, so a call was repeated many more times than the maas
  client asked for, and a switchover took correspondingly longer to surface.
  `WithHttpClient` now states the two settings a replacement has to keep: no
  retries of its own, and no client-wide timeout, because the same client serves
  the topic watch long poll.
