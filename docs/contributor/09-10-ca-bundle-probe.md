# CA Bundle Probe

## Overview

The CA bundle probe is a periodic background job that checks whether the TLS certificate of the SAP Service Manager token URL is trusted by the CA bundle currently mounted on the BTP Manager Pod.

It is designed for Kyma clusters where the `rt-bootstrapper` module is active. In such clusters, a custom CA bundle is injected into Pods using a volume mount named `rt-bootstrapper-certs`. The probe detects this mount and uses the custom bundle as the certificate pool for TLS verification.

The probe is disabled by default (`ProbeInterval: 0s`). It requires a probe image to be configured using the **PROBE_IMAGE** environment variable.

## How It Works

BTP Manager runs a `ProbeRunner` as a controller-runtime `Runnable`. On each interval tick, it performs the following steps:

1. Deletes any leftover probe Job from a previous cycle.
2. Creates a new Kubernetes Job (`btp-manager-ca-bundle-probe`) in `kyma-system`.
3. Waits for the Job to complete (up to 5 minutes).
4. Reads the results from annotations on the `BtpOperator` custom resource (CR).
5. Updates the `btpmanager_credential_probe_status` Prometheus metric.
6. If the CA bundle hash changed and TLS is healthy, restarts the `sap-btp-operator` Pods.
7. Updates the `tls-probe-last-hash` annotation on the `BtpOperator` CR.

## Probe Job

The probe Job runs a single container using the image configured with **PROBE_IMAGE**. The Job performs the following steps:

- Runs with `RestartPolicy: Never` and `BackoffLimit: 0`.
- Runs with Istio sidecar injection disabled (`sidecar.istio.io/inject: "false"`).
- Uses the `btp-manager-ca-bundle-probe` ServiceAccount.
- Writes results as annotations on the `BtpOperator` CR, then exits.

### Annotations Written by the Probe

| Annotation | Values | Description |
|---|---|---|
| `tls-probe-status` | `ok`, `alert`, or empty | TLS verification result |
| `tls-probe-hash` | SHA256 hex string | Hash of the CA bundle file |
| `tls-probe-updated-at` | RFC3339 timestamp | Time the probe wrote its results |

### Annotation Managed by BTP Manager

| Annotation | Description |
|---|---|
| `tls-probe-last-hash` | Hash from the previous cycle, used to detect CA bundle rotation |

## Decision Logic

After each Job completes, BTP Manager reads the probe annotations and acts as follows. "No action" means BTP Manager takes no action; the probe Job itself may log internally.

| Mount present | TLS result | Hash changed | BTP Manager action |
|---|---|---|---|
| No | ok | n/a | No action (public landscape, all good) |
| No | failed (x509) | n/a | No action (probe logs internally; no annotation written) |
| Yes | ok | No | No action (TLS healthy, no rotation) |
| Yes | ok | Yes | Restart `sap-btp-operator` Pods |
| Yes | failed (x509) | Any | Alert metric set to 1 |
| Any | failed (other) | Any | No action (probe logs internally; no annotation written) |

Mount detection is based solely on the presence of the `rt-bootstrapper-certs` volume mount on the BTP Manager Pod (using **POD_NAME** environment variable).

## Configuration

| Parameter | Source | Default | Description |
|---|---|---|---|
| **ProbeInterval** | ConfigMap `sap-btp-manager` / CLI flag `--probe-interval` | `0s` (disabled) | How often to run the probe. Set to `0` to disable. |
| **PROBE_IMAGE** | Environment variable | None | Container image for the probe Job. Required to enable the probe. |
| **PROBE_TOKENURL_OVERRIDE** | Environment variable | None | Override the token URL used by the probe (for testing). |
| **PROBE_FORCE_HASH** | Environment variable | None | Force a specific hash value (for testing). |

## Metric

| Metric | Type | Description |
|---|---|---|
| `btpmanager_credential_probe_status` | Gauge | `1` when alert (CA mounted but cert not trusted). Set to `0` when the probe writes a non-alert result. **Not updated** on silent-exit cycles (no mount + TLS ok), so the gauge retains its last written value until the next cycle, where the probe writes annotations. |

## RBAC

The probe Job Pod uses the `btp-manager-ca-bundle-probe` ServiceAccount, which has the following permissions:
- `get` and `patch` on `btpoperators.operator.kyma-project.io` (to write result annotations)
- `get` on `secrets` in `kyma-system` (to read the CA bundle)

BTP Manager itself requires additional RBAC to manage the probe Jobs:
- `get`, `list`, `watch`, `create`, `delete` on `batch/jobs`
