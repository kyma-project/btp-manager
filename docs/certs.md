---
title: Certification management
---

![Certification management diagram](./assets/certs.svg)

The reconciliation of certifications can be triggered from the three events, first is normal scheduled reconciliation, second event is by doing edit on btpOperator CR, and the third is by use of custom watchers founded on Secrets and Webhooks resources.
BTP Manager maintains two Secrets, `ca-server-cert` and `webhook-server-cert`, which are used to allow communication within BTP Operator Webhooks and thus allow the creation of resources like ServiceInstances and ServiceBindings.
During provisioning, at first, `ca-server-cert` is created. It is a self-signed CA certificate. Then based on that, the application creates a signed cert, `webhook-server-cert`, which is mounted under the deployment.
Webhooks have a caBundle field set to the content of `ca-server-cert,` and BTP Manager manages this field.
The `ca-server-cert`, `webhook-server-cert`, and Webhooks' caBundles are kept in sync by using the reconciliation mechanism, which means every manual change in these resources by the user will automatically trigger regeneration of all the three resources.
BTP Manager maintains the resources by creating, deleting, and updating actions during the reconciliation. The goal is to keep `ca-server-cert`, `webhook-server-cert`, and Webhooks' caBundle in sync all the time.
Scheduled reconciliation also checks expiration dates of certificate's, and if it detects that a certificate expires soon, it regenerates it in advance so that the processes run smoothly.

