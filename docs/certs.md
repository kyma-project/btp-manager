![Certification management diagram](./assets/certs.svg)

## Certification management

BTP Manager maintain two secrets ca-server-cert and webhook-server-cert which are used to allow communication within webhooks, and in result allow to create resources like ServiceInstances and ServiceBindings
BTP Manager maintain resources during reconcilation, which mean create, delete and update. The goal is to keep in sync ca-server-cert, webhook-servert-cert and Webhooks.
