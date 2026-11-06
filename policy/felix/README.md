# Felix watch bookmark backport

This alternative keeps Calico v3.24.5 at f1a1611acb98d9187f48bbbe2227301aa69f0499 and retains the existing Go/dependency updates. It adds bookmark support without adding resource subscriptions, CRDs, RBAC, or a new Calico base version.

The alternative full-version upgrade is on the codex/felix-bookmark-v330 branch.

## Patch sources

The existing patch order is retained. Separate upstream changes are appended:

| Patch | Upstream commit | Purpose |
| --- | --- | --- |
| 0008 | bf7146fc171775729224a7899b5337518a408604 | Opt-in WatchOptions for the existing backend resource types. |
| 0009 | 405f7f436085e6be45086e81c7d0d91673357798 | Bookmark requests, conversion, profile revisions, and syncer revision tracking. |
| 0010 | 508aedb9f343a7f3ae5f4dbb4cd9c66ab10de112 | Profile handling and watch recovery corrections. |
| 0011 | 07163f8a4b8ab851706d95bd7ad3d6bc7ebfd031 | Expired/too-new RV and refused/throttled connection recovery. |
| 0012 | Additional regression coverage | Bookmark options, conversion, and composite profile revisions. |
| 0013 | 2f8942443fb7d0385ae18adea74f04b6a34a6aec | Recovery threshold and matching upstream reconnect/error tests. |

Contexts are adapted to v3.24.5. Changes for resource types absent from that release are omitted. Each upstream-derived patch records its full original commit ID. Public client watches keep bookmarks disabled unless the internal watcher explicitly requests them.

## Validation

- Built with Go 1.26.10 and CGO_ENABLED=0 on Linux/arm64.
- Cross-compiled the Felix executable for Linux/amd64.
- Re-applied the complete numbered patch set to a clean upstream source archive using the same git apply command as the policy Dockerfile.
- Bookmark regression coverage checks opt-in request options, revision-only event conversion, combined Namespace/ServiceAccount profile revisions, and resuming a watch from the latest bookmark.
- The targeted Ginkgo runs exclude the Datastore integration cases, which require a Kubernetes test cluster, and the upstream reconnect-duration assertion with a 750ms upper bound. That duration assertion also fails on an unmodified v3.30.0 in the local test environment; it is retained unchanged in the patches.
- The repository make test-quick target was run in a Linux/arm64 container. It did not pass: failures include host-path/TC/device-dependent cases and gomonkey-based tests outside Felix. Those packages are not changed by these policy-input branches.
- No policy image was published, no consuming image reference was changed, and no customer-cluster or datapath end-to-end validation was performed. These branches contain build inputs for review.

## Resource versions

The old performance patch added resourceVersion=0 to functional-test List calls. Those additions are removed. The upstream syncer's initial and full-resync RV=0 behavior is retained. This change does not claim that List pagination becomes effective on every Kubernetes server version.

