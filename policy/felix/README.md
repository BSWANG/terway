# Felix v3.30.0 upgrade alternative

This alternative upgrades the Calico base from v3.24.5 to v3.30.0 at 572a32dd9b2f9784d66d74712d5ec2e703d25083. Watch bookmark support is provided by the upstream version.

The smaller alternative retains v3.24.5 and backports the relevant commits on the codex/felix-bookmark-backport branch.

## Patch scope

1. Preserve Terway configuration ownership, workload interface sysctl ownership, outgoing NAT behavior, and RHEL kernel feature detection.
2. Preserve Kubernetes client rate limits without the old FV List RV=0 additions.
3. Preserve Aliyun Pod IP annotation conversion and cover IPv4, dual-stack, status precedence, and invalid annotations.
4. Keep exactly the pre-upgrade Felix resource subscription set. Do not add Tier, staged policy, BGPPeer, AdminNetworkPolicy, or BaselineAdminNetworkPolicy subscriptions. Existing deployments therefore do not need new CRDs or RBAC for this alternative.
5. Retain main's dependency version floors for overlapping modules, while retaining the v3.30.0 Kubernetes dependency family. The dependency graph is resolved with Go 1.26.10.
6. Replace Kubernetes core-validation imports in flow-name validation with the exact apimachinery functions those core validators alias. This avoids pulling incompatible apiserver storage code into the Felix binary with the updated OpenTelemetry dependencies. The v3.30.0 proxy interfaces remain unchanged.
7. Cover bookmark request options, revision-only conversion, and composite profile revisions.

The Cilium build stage and consuming policy-image references are unchanged. This is an alternative to the backport branch; do not combine their Felix patch directories.

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

The upgrade-specific targeted checks also cover kernel feature detection, Pod conversion, and goldmane/proto name-validation behavior.

