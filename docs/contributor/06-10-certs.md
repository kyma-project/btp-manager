# Certification management

BTP Manager maintains two Secrets, `ca-server-cert` and `webhook-server-cert`, which are used to allow for communication within BTP Operator webhooks and thus allow for the creation of resources like ServiceInstances and ServiceBindings. The `ca-server-cert`, `webhook-server-cert`, and their CA Bundles are kept in sync by using the reconciliation mechanism, which means that every manual change in these resources the user makes automatically triggers the regeneration of all three resources.

BTP Manager maintains the resources by creating, deleting, and updating actions during the reconciliation. The goal is to keep `ca-server-cert`, `webhook-server-cert`, and their CA Bundle in sync all the time. The scheduled reconciliation also checks the certificates’ expiration dates, and if it detects that a certificate expires soon, it regenerates it in advance so that the processes run smoothly.

![Certification management diagram](../assets/certs.svg)

1.	Certification reconciliation is triggered by one of the three events: scheduled reconciliation, editing [BtpOperator CR](/api/v1alpha1/btpoperator_types.go), or using custom watchers founded on Secret and Webhook resources. 
2.	First, during provisioning, BTP Manager checks if a self-signed CA certificate, `ca-server-cert`, exists. If it doesn't exist:  
    a.	BTP Manager generates the certificate.  
    b.	Based on that, the application creates a signed cert, `webhook-server-cert`, which is mounted under the deployment.  
    c.	The webhooks have a CA Bundle field set to the content of `ca-server-cert,` and BTP Manager manages this field; certificates' reconciliation is complete.  
3.	If the `ca-server-cert` Secret exists, BTP Manager checks if the `webhook-server-cert` Secret exists. If not, it is created as described in step 2b, and then step 2c follows. The process of certificates' reconciliation is complete.
4.	The webhooks have a CA Bundle field set to the content of the `ca-server-cert` Secret, and BTP Manager manages this field. If `webhook-server-cert` exists, BTP Manager checks if the current webhook CA Bundle is the same as the `ca-server-cert` Secret. If it is different, BTP Manager recreates `ca-server-cert` as described in step 2a. Then the procedure progresses as described in steps 2b and 2c until the process of certificates' reconciliation is complete.
5.	If the current webhook CA Bundle is the same as the `ca-server-cert` Secret, BTP Manager checks if `webhook-server-cert` is signed by `ca-server-cert`. If not signed, BTP Manager recreates `ca-server-cert` as described in step 2a. Then the procedure progresses as described in steps 2b and 2c until the process of certificates' reconciliation is complete.
6.	The scheduled reconciliation checks the expiration date of ` ca-server-cert`. If it detects that the certificate expires soon, it regenerates `ca-server-cert` as described in point 2a. Then the procedure progresses as described in steps 2b and 2c until the process of certificates' reconciliation is complete.
7.	If `ca-server-cert` is still valid, the scheduled reconciliation checks the expiration date of `webhook-server-cert`. If it detects that the certificate expires soon, it recreates the `webhook-server-cert` Secret. The process continues as described in points 2b and 2c.
8.	The process of certificates reconciliation is complete.
