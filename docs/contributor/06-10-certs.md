# Certification management

![Certification management diagram](../assets/certs.svg)

Certification reconciliation is triggered by one of the three events: scheduled reconciliation, editing [BtpOperator CR](../../api/v1alpha1/btpoperator_types.go), or using custom watchers founded on Secret and Webhook resources.

BTP Manager maintains two Secrets, `ca-server-cert` and `webhook-server-cert`, which are used to allow for communication within BTP Operator webhooks and thus allow for the creation of resources like ServiceInstances and ServiceBindings.
First, during provisioning, `ca-server-cert` is created. It is a self-signed CA certificate. Then, based on that, the application creates a signed cert, `webhook-server-cert`, which is mounted under the deployment.

The webhooks have a CA Bundle field set to the content of `ca-server-cert,` and BTP Manager manages this field.
The `ca-server-cert`, `webhook-server-cert`, and their CA Bundles are kept in sync by using the reconciliation mechanism, which means every manual change in these resources that the user makes automatically triggers the regeneration of all three resources.

BTP Manager maintains the resources by creating, deleting, and updating actions during the reconciliation. The goal is to keep `ca-server-cert`, `webhook-server-cert`, and their CA Bundle in sync all the time.
The scheduled reconciliation also checks the certificate's expiration dates, and if it detects that a certificate expires soon, it regenerates it in advance so that the processes run smoothly.
