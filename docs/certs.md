![Certification management diagram](./assets/certs.svg)

## Certification management

BTP Manager maintain two secrets `ca-server-cert` and `webhook-server-cert` which are used to allow communication within BTP Operator webhooks, and in result allow to create resources like ServiceInstances and ServiceBindings
During provisioning, first there is created `ca-server-cert` which is self sign CA certificate, then based on that, application create signed cert `webhook-server-cert`, which is mounted under the deployment.
Webhooks have CABundle field which is set to the content of ca-server-cert and this field is managed by BTP Manager.
The `ca-server-cert`, `webhook-server-cert`, and webhooks caBundles are keep in sync by using reconcilation mechanism, which mean every manual change by user in this resources will trigger automatically regeneration of all three resources.
BTP Manager maintain resources by creating, deleting and updating actions during reconcilation. The goal is to keep `ca-server-cert`, `webhook-servert-cert` and webhooks ca bundles in sync, all the time.
Scheduled reconcilation check also expiration dates of certificates and if it detected that certificate expires soon, it regenerates it in advanced, to keep things smoothly.