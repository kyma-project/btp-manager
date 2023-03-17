---
title: Certification management
---

![Certification management diagram](./assets/certs.svg)

BTP Manager maintains two secrets, `ca-server-cert` and `webhook-server-cert`, which are used to allow communication within BTP Operator webhooks and thus allow the creation of resources like ServiceInstances and ServiceBindings.
During provisioning, at first, `ca-server-cert` is created. It is a self-sign CA certificate. Then based on that, the application creates a signed cert, `webhook-server-cert`, which is mounted under the deployment.
Webhooks have a caBundle field set to the content of `ca-server-cert,` and BTP Manager manages this field.
The `ca-server-cert`, `webhook-server-cert`, and webhooks' caBundles are kept in sync by using the reconciliation mechanism, which means every manual change in these resources by the user will automatically trigger regeneration of all three resources.
BTP Manager maintains resources by creating, deleting, and updating actions during reconciliation. The goal is to keep `ca-server-cert`, `webhook-server-cert`, and webhooks' caBundle in sync all the time.
Scheduled reconciliation also checks expiration dates of certificates, and if it detects that a certificate expires soon, it regenerates it in advance so that the processes run smoothly.