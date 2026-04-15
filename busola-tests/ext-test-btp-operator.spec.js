/// <reference types="cypress" />
import jsyaml from 'js-yaml';
import 'cypress-file-upload';
import config from '../config';

const EXTENSION_NAME = 'btpoperator-module';
const EXTENSION_YAML_PATH = 'sap-btp-operator-extension.yaml';

context('Test BTP Operator extension', () => {
  Cypress.skipAfterFail();

  before(() => {
    cy.handleExceptions();

    cy.loginAndSelectCluster({
      fileName: 'kubeconfig.yaml',
      expectedLocation: /overview$/,
    });

    // Create mock sap-btp-manager secret (needed by BTP Manager Secret panel)
    cy.contains('ui5-button', 'Upload YAML').click();

    const btpManagerSecretYaml = `apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: sap-btp-manager
  namespace: kyma-system
  labels:
    app.kubernetes.io/managed-by: kcp-kyma-environment-broker
data:
  clientid: bW9jay1jbGllbnQtaWQ=
  clientsecret: bW9jay1jbGllbnQtc2VjcmV0
  sm_url: aHR0cHM6Ly9zZXJ2aWNlLW1hbmFnZXIuY2ZhcHBzLm1vY2suaGFuYS5vbmRlbWFuZC5jb20=
  tokenurl: aHR0cHM6Ly9tb2NrLmF1dGhlbnRpY2F0aW9uLnNhcC5oYW5hLm9uZGVtYW5kLmNvbS9vYXV0aC90b2tlbg==
  cluster_id: bW9jay1rM2QtY2x1c3Rlci1pZC0xMjM0NQ==`;

    cy.pasteToMonaco(btpManagerSecretYaml);

    cy.get('ui5-dialog')
      .contains('ui5-button', 'Upload')
      .should('be.visible')
      .click();

    cy.wait(1000);

    cy.get('ui5-dialog[header-text="Upload YAML"]').within(() => {
      cy.contains('ui5-button', 'Close').should('be.visible').click();
    });

    cy.wait(1000);
  });

  it('Upload extension ConfigMap', () => {
    // Go to Cluster Overview
    cy.getLeftNav().contains('Cluster Overview').click();

    // Upload extension ConfigMap
    cy.contains('ui5-button', 'Upload YAML').click();

    cy.loadFiles(EXTENSION_YAML_PATH).then((resources) => {
      const configMap = jsyaml.dump(resources[0]);
      cy.pasteToMonaco(configMap);
    });

    cy.get('ui5-dialog')
      .contains('ui5-button', 'Upload')
      .should('be.visible')
      .click();

    cy.get('ui5-dialog[header-text="Upload YAML"]')
      .find('.status-message-success')
      .should('have.length', 1);

    cy.get('ui5-dialog[header-text="Upload YAML"]').within(() => {
      cy.contains('ui5-button', 'Close').should('be.visible').click();
    });

    cy.wait(1000);

    // Upload mock sap-btp-service-operator secret (needed for SAP BTP Service Operator Secret panel)
    cy.contains('ui5-button', 'Upload YAML').click();

    const operatorSecretYaml = `apiVersion: v1
kind: Secret
metadata:
  name: sap-btp-service-operator
  namespace: kyma-system
type: Opaque
data:
  clientid: bW9jay1jbGllbnRpZA==`;

    cy.pasteToMonaco(operatorSecretYaml);

    cy.get('ui5-dialog')
      .contains('ui5-button', 'Upload')
      .should('be.visible')
      .click();

    cy.wait(1000);

    cy.get('ui5-dialog[header-text="Upload YAML"]').within(() => {
      cy.contains('ui5-button', 'Close').should('be.visible').click();
    });

    cy.wait(1000);

    // Navigate to kyma-system and open BTP Operators
    cy.getLeftNav().contains('Namespaces').click();
    cy.contains('ui5-link', 'kyma-system').click();

    cy.getLeftNav().contains('Kyma').should('be.visible').click();
    cy.getLeftNav().contains('BTP Operators').should('be.visible').click();

    cy.contains('BTP Operators').should('be.visible');

    cy.clickGenericListLink('btpoperator');
    cy.wait(1000);

    // --- Verify header (Metadata card) items ---
    cy.getMidColumn().within(() => {
      cy.contains('Documentation').should('be.visible');
      cy.contains('SAP BTP Operator Module').should('be.visible');
      cy.contains('Service Instances').should('be.visible');
      cy.contains('Service Bindings').should('be.visible');
    });

    // --- Verify BTP Operator Secrets panel ---
    cy.getMidColumn().within(() => {
      cy.contains('BTP Operator Secrets').should('be.visible');

      // BTP Manager Secret sub-panel
      cy.contains('BTP Manager Secret').should('be.visible');
      cy.contains('Managed').should('be.visible');
      cy.contains('Credentials Namespace').should('be.visible');
      cy.contains('kyma-system').should('be.visible');

      // SAP BTP Service Operator Secret sub-panel
      cy.contains('SAP BTP Service Operator Secret').should('be.visible');
      cy.contains('Inherited').should('be.visible');
    });

    // --- Test Edit ResourceLink → navigates to sap-btp-manager secret ---
    cy.getMidColumn().within(() => {
      cy.contains('ui5-link', 'Edit').click();
    });
    cy.wait(500);
    cy.contains('sap-btp-manager').should('be.visible');
    cy.go('back');
    cy.wait(500);

    // --- Test Service Instances count link → ServiceInstance CRD page ---
    cy.getMidColumn().within(() => {
      cy.contains('Service Instances')
        .parent()
        .find('ui5-link')
        .first()
        .click();
    });
    cy.wait(500);
    cy.contains('serviceinstances.services.cloud.sap.com').should('be.visible');
    cy.contains('Resource Details').should('be.visible');
    cy.go('back');
    cy.wait(500);

    // --- Test Service Bindings count link → ServiceBinding CRD page ---
    cy.getMidColumn().within(() => {
      cy.contains('Service Bindings')
        .parent()
        .find('ui5-link')
        .first()
        .click();
    });
    cy.wait(500);
    cy.contains('servicebindings.services.cloud.sap.com').should('be.visible');
    cy.contains('Resource Details').should('be.visible');
    cy.go('back');
    cy.wait(500);
  });

  it('Configure custom credentials namespace', () => {
    cy.getLeftNav().contains('Cluster Overview').click();

    // 1. Create test namespace
    cy.createNamespace('test');

    // 2. Create namespace-based secret in test namespace
    cy.navigateTo('Configuration', 'Secrets');

    cy.contains('ui5-button', 'Create').click();

    cy.get('[accessible-name="Secret name"]:visible')
      .find('input')
      .type('test-sap-btp-service-operator', { force: true });

    cy.get('[data-testid="create-form-footer-bar"]')
      .contains('ui5-button:visible', 'Create')
      .click();

    cy.wait(2000);
    cy.contains('test-sap-btp-service-operator').should('be.visible');

    // 3. Navigate to kyma-system and edit sap-btp-manager secret
    cy.getLeftNav().contains('Cluster Overview').click();
    cy.wait(1000);
    cy.reload();
    cy.wait(1000);

    cy.getLeftNav().contains('Namespaces').click();
    cy.contains('ui5-link', 'kyma-system').click();
    cy.wait(1000);
    cy.getLeftNav().contains('Secrets').click();

    cy.clickGenericListLink('sap-btp-manager');
    cy.wait(500);

    cy.inspectTab('Edit');

    // Add skip-reconciliation label
    cy.contains('Labels').click();

    cy.get('[placeholder="Enter value"]:visible')
      .filter((index, el) => !el.value && el.getAttribute('accessible-name') === 'Labels value')
      .first()
      .find('input, textarea')
      .first()
      .type('true', { force: true });

    cy.get('[placeholder="Enter key"]:visible')
      .filter((index, el) => !el.value && el.getAttribute('accessible-name') === 'Labels key')
      .first()
      .find('input')
      .type('kyma-project.io/skip-reconciliation', { force: true });

    cy.saveChanges('Edit');
    cy.wait(2000);

    // Add credentials_namespace data field
    cy.get('[placeholder="Enter value"]:visible')
      .filter((index, el) => !el.value && el.getAttribute('accessible-name') === 'Data value')
      .first()
      .find('input, textarea')
      .first()
      .type('test', { force: true });

    cy.get('[placeholder="Enter key"]:visible')
      .filter((index, el) => !el.value && el.getAttribute('accessible-name') === 'Data key')
      .first()
      .find('input')
      .type('credentials_namespace', { force: true });

    cy.saveChanges('Edit');
    cy.wait(5000);

    // 4. Navigate back to BTP Operator and verify changes
    cy.getLeftNav().contains('BTP Operators').click();
    cy.clickGenericListLink('btpoperator');
    cy.wait(1000);

    cy.getMidColumn().within(() => {
      // Credentials Namespace now shows "test"
      cy.contains('Credentials Namespace').should('be.visible');
      cy.contains('test').should('be.visible');

      // BTP Manager Secret shows Unmanaged badge
      cy.contains('Unmanaged').should('be.visible');

      // Namespace-Based Secrets shows test secret as "In Use"
      cy.contains('Namespace-Based Secrets').scrollIntoView();
      cy.wait(500);
      cy.contains('test-sap-btp-service-operator').scrollIntoView();
      cy.contains('In Use').should('exist');
    });

    // 5. Create Service Instance and Service Binding
    cy.getLeftNav().contains('Cluster Overview').click();
    cy.wait(1000);

    cy.contains('ui5-button', 'Upload YAML').click();

    const serviceInstanceAndBindingYaml = `apiVersion: services.cloud.sap.com/v1
kind: ServiceInstance
metadata:
  name: test-service-instance
  namespace: kyma-system
spec:
  serviceOfferingName: test-offering
  servicePlanName: test-plan
---
apiVersion: services.cloud.sap.com/v1
kind: ServiceBinding
metadata:
  name: test-service-binding
  namespace: kyma-system
spec:
  serviceInstanceName: test-service-instance`;

    cy.pasteToMonaco(serviceInstanceAndBindingYaml);

    cy.get('ui5-dialog')
      .contains('ui5-button', 'Upload')
      .should('be.visible')
      .click();

    cy.wait(3000);

    cy.get('ui5-dialog[header-text="Upload YAML"]').within(() => {
      cy.contains('ui5-button', 'Close').should('be.visible').click();
    });

    cy.wait(1000);

    // 6. Navigate back and verify header counts updated
    cy.getLeftNav().contains('Namespaces').click();
    cy.contains('ui5-link', 'kyma-system').click();
    cy.getLeftNav().contains('BTP Operators').click();
    cy.clickGenericListLink('btpoperator');
    cy.wait(1000);

    cy.getMidColumn().within(() => {
      cy.contains('Service Instances').should('be.visible');
      cy.contains('Service Bindings').should('be.visible');
    });
  });

  it('Custom Secrets shows referenced secrets with correct status', () => {
    // Create a ServiceInstance referencing a custom secret
    cy.getLeftNav().contains('Cluster Overview').click();
    cy.wait(1000);

    cy.contains('ui5-button', 'Upload YAML').click();

    const customSecretInstanceYaml = `apiVersion: services.cloud.sap.com/v1
kind: ServiceInstance
metadata:
  name: test-custom-secret-instance
  namespace: kyma-system
spec:
  serviceOfferingName: test-offering
  servicePlanName: test-plan
  btpAccessCredentialsSecret: test-custom-secret`;

    cy.pasteToMonaco(customSecretInstanceYaml);

    cy.get('ui5-dialog')
      .contains('ui5-button', 'Upload')
      .should('be.visible')
      .click();

    cy.wait(3000);

    cy.get('ui5-dialog[header-text="Upload YAML"]').within(() => {
      cy.contains('ui5-button', 'Close').should('be.visible').click();
    });

    cy.wait(1000);

    // Navigate to btpoperator details
    cy.getLeftNav().contains('Namespaces').click();
    cy.contains('ui5-link', 'kyma-system').click();
    cy.getLeftNav().contains('BTP Operators').click();
    cy.clickGenericListLink('btpoperator');
    cy.wait(1000);

    cy.getMidColumn().within(() => {
      // Custom Secrets panel visible
      cy.contains('Custom Secrets').scrollIntoView();
      cy.wait(500);

      // Secret row appears in table
      cy.contains('test-custom-secret').scrollIntoView();
      cy.contains('test-custom-secret').should('exist');

      // Not in Use: kyma-system ≠ managedNamespace "test" (set in previous test)
      cy.contains('Not in Use').should('exist');

      // Service Instances count shows "1" in the same row
      cy.contains('test-custom-secret')
        .closest('ui5-table-row')
        .contains('1')
        .should('exist');
    });
  });
});
