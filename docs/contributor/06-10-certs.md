# Certification Management

## Overview

BTP Manager maintains two Secrets, `ca-server-cert` and `webhook-server-cert`. They are used for communication within the SAP BTP service operator's webhooks and for creating resources, such as ServiceInstances and ServiceBindings. The [reconciliation mechanism](#reconciliation-mechanism) synchronizes the two Secrets and their CA Bundles, which means that whenever the user manually changes them, they are automatically regenerated.

BTP Manager maintains the resources by creating, deleting, and updating them during the reconciliation. The goal is to keep `ca-server-cert`, `webhook-server-cert`, and the webhooks' CA Bundles synchronized at all times. The reconciliation also checks the certificates’ expiration dates. If it detects that a certificate is about to expire, it regenerates it in advance to ensure the processes run smoothly.

## Reconciliation Mechanism

![Certification management diagram](../assets/certs.drawio.svg)

1.	Certification reconciliation is triggered by one of the three events: scheduled reconciliation, editing [BtpOperator custom resource (CR)](/api/v1alpha1/btpoperator_types.go), or using custom watchers founded on the Secret and webhook resources. 
2.	During provisioning, BTP Manager checks if a self-signed CA certificate, `ca-server-cert`, exists. If it doesn't exist, the following steps are performed:  
    a.	BTP Manager generates the certificate.  
    b.	Based on that, the application creates a signed certificate, `webhook-server-cert`, which is mounted under the Deployment.  
    c.	The webhooks have a **caBundle** field set to the content of `ca-server-cert`, and BTP Manager manages this field; the process of certificates' reconciliation is complete.  
3.	If the `ca-server-cert` Secret exists, BTP Manager checks if the `webhook-server-cert` Secret exists. If not, it is created as described in step 2b, and then step 2c follows. The process of certificates' reconciliation is complete.
4.	If `webhook-server-cert` exists, BTP Manager checks if the current webhook CA Bundle is the same as the `ca-server-cert` Secret. If it is different, BTP Manager recreates `ca-server-cert` as described in step 2a. Then the procedure progresses as described in steps 2b and 2c until the process of certificates' reconciliation is complete.
5.	If the current webhook CA Bundle is the same as the `ca-server-cert` Secret, BTP Manager checks if `webhook-server-cert` is signed by `ca-server-cert`. If not signed, BTP Manager recreates `ca-server-cert` as described in step 2a. Then the procedure progresses as described in steps 2b and 2c until the process of certificates' reconciliation is complete.
6.	The scheduled reconciliation checks the expiration date of `ca-server-cert`. If it detects that the certificate expires soon, it regenerates `ca-server-cert` as described in point 2a. Then the procedure progresses as described in steps 2b and 2c until the process of certificates' reconciliation is complete.
7.	If `ca-server-cert` is still valid, the scheduled reconciliation checks the expiration date of `webhook-server-cert`. If it detects that the certificate expires soon, it recreates the `webhook-server-cert` Secret. The process continues as described in points 2b and 2c.
8.	The process of certificates' reconciliation is complete.
