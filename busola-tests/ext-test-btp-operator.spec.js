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
    
    // Login and select cluster using kubeconfig
    cy.loginAndSelectCluster({
      fileName: 'kubeconfig.yaml',
      expectedLocation: /overview$/,
    });
  });

  it('Upload extension ConfigMap', () => {
    // Go to Cluster Overview
    cy.getLeftNav().contains('Cluster Overview').click();

    // Use Upload YAML instead of Create form
    cy.contains('ui5-button', 'Upload YAML').click();

    // Load and paste extension ConfigMap
    cy.loadFiles(EXTENSION_YAML_PATH).then((resources) => {
      const configMap = jsyaml.dump(resources[0]);
      cy.pasteToMonaco(configMap);
    });

    // Upload
    cy.get('ui5-dialog')
      .contains('ui5-button', 'Upload')
      .should('be.visible')
      .click();

    // Verify success
    cy.get('ui5-dialog[header-text="Upload YAML"]')
      .find('.status-message-success')
      .should('have.length', 1);

    // Close dialog
    cy.get('ui5-dialog[header-text="Upload YAML"]').within(() => {
      cy.contains('ui5-button', 'Close').should('be.visible').click();
    });

    cy.wait(1000);
    
    // Verify extension UI is available - navigate to kyma-system and check BTP Operators menu
    cy.getLeftNav().contains('Namespaces').click();
    cy.contains('ui5-link', 'kyma-system').click();
    

    // Check if BTP Operators menu item exists (extension loaded)
    cy.getLeftNav().contains('Kyma').should('be.visible').click();
    cy.getLeftNav().contains('BTP Operators').should('be.visible').click();
    
    // Verify we're on the BTP Operators page
    cy.contains('BTP Operators').should('be.visible');
    
    // Click on btpoperator to open details
    cy.clickGenericListLink('btpoperator');
    cy.wait(1000);
    
    // Verify default state of BTP Operator extension
    cy.getMidColumn().within(() => {
      
      // Verify Credentials Namespace is kyma-system
      cy.contains('Credentials Namespace').should('be.visible');
      cy.contains('kyma-system').should('be.visible');
      
      // Verify Service Instances section shows 0 items
      cy.contains('Service Instances and Bindings').should('be.visible');
      cy.contains('Service Instances').should('be.visible');
      cy.contains('0 item/s').should('be.visible');
      
      // Verify SAP BTP Manager Secret section exists
      cy.contains('SAP BTP Manager Secret').should('be.visible');
      
      // Verify secret is managed by Kyma message
      cy.contains('This Secret is managed by Kyma').should('be.visible');
    });
    
    // Test all internal links
    // 1. Test sap-btp-manager secret link
    cy.getMidColumn().within(() => {
      cy.get('ui5-link').contains('sap-btp-manager').click();
    });
    cy.wait(500);
    cy.contains('sap-btp-manager').should('be.visible');
    cy.go('back');
    cy.wait(500);
    
    // 2. Test Service Instances "See details" link
    cy.getMidColumn().within(() => {
      cy.contains('Service Instances').should('be.visible');
      cy.get('ui5-link').contains('See details').first().click();
    });
    cy.wait(500);
    cy.contains('serviceinstances.services.cloud.sap.com').should('be.visible');
    cy.contains('Resource Details').should('be.visible');
    cy.go('back');
    cy.wait(500);
    
    // 3. Test Service Bindings "See details" link - use second "See details" link
    cy.getMidColumn().within(() => {
      cy.contains('Service Bindings').scrollIntoView();
      cy.wait(300);
      // Get all "See details" links and click the second one
      cy.get('ui5-link').then($links => {
        const seeDetailsLinks = $links.filter((i, el) => el.textContent.includes('See details'));
        cy.wrap(seeDetailsLinks[1]).click({ force: true });
      });
    });
    cy.wait(500);
    cy.contains('servicebindings.services.cloud.sap.com').should('be.visible');
    cy.contains('Resource Details').should('be.visible');
    cy.go('back');
    cy.wait(500);
  });

  it('Configure custom credentials namespace', () => {
    // Go back to Cluster Overview first
    cy.getLeftNav().contains('Cluster Overview').click();
    
    // 1. Create test namespace
    cy.createNamespace('test');
    
    // 2. Create secret in test namespace (we're already in it after creation)
    cy.navigateTo('Configuration', 'Secrets');
    
    cy.contains('ui5-button', 'Create').click();
    
    // Fill secret name
    cy.get('[accessible-name="Secret name"]:visible')
      .find('input')
      .type('test-sap-btp-service-operator', { force: true });
    
    // Create empty secret (no data needed)
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
    
    // Switch to Edit tab using inspectTab
    cy.inspectTab('Edit');
    
    // Expand Labels section
    cy.contains('Labels').click();

    // Add label value - find corresponding value field
    cy.get('[placeholder="Enter value"]:visible')
      .filter((index, el) => !el.value && el.getAttribute('accessible-name') === 'Labels value')
      .first()
      .find('input, textarea')
      .first()
      .type('true', { force: true });

    // Now add label - find the first empty key field in Labels
    cy.get('[placeholder="Enter key"]:visible')
      .filter((index, el) => !el.value && el.getAttribute('accessible-name') === 'Labels key')
      .first()
      .find('input')
      .type('kyma-project.io/skip-reconciliation', { force: true });

    cy.saveChanges('Edit');
    cy.wait(2000);

    // Add data value
    cy.get('[placeholder="Enter value"]:visible')
      .filter((index, el) => !el.value && el.getAttribute('accessible-name') === 'Data value')
      .first()
      .find('input, textarea')
      .first()
      .type('test', { force: true });
      
    // Now add data field - find the first empty key field in Data
    cy.get('[placeholder="Enter key"]:visible')
      .filter((index, el) => !el.value && el.getAttribute('accessible-name') === 'Data key')
      .first()
      .find('input')
      .type('credentials_namespace', { force: true });

    // Save the secret immediately
    cy.saveChanges('Edit');
    
    cy.wait(5000);
    
    // 4. Navigate back to BTP Operator and verify changes
    cy.getLeftNav().contains('BTP Operators').click();
    cy.clickGenericListLink('btpoperator');
    cy.wait(1000);
    
    // Verify Credentials Namespace is now "test"
    cy.getMidColumn().within(() => {
      cy.contains('Credentials Namespace').should('be.visible');
      cy.contains('test').should('be.visible');
    });
    
    // Scroll down to Namespaced Secrets section
    cy.getMidColumn().within(() => {
      cy.contains('Namespaced Secrets').scrollIntoView();
      cy.wait(500);
      
      // Scroll to the secret name to ensure it's visible
      cy.contains('test-sap-btp-service-operator').scrollIntoView();
      
      // Verify "In Use" badge exists in the same row
      cy.contains('In Use').should('exist');
      
      // Verify secret is now manually managed message (after skip-reconciliation label)
      cy.contains('This Secret is NOT managed by Kyma').should('be.visible');
    });
    
    // 5. Create Service Instance and Service Binding using Upload YAML
    // Go to Namespaces and select test namespace
    cy.getLeftNav().contains('Cluster Overview').click();
    cy.wait(1000);
    
    // Use Upload YAML from namespace view
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
    
    // Wait for upload to complete (may have errors due to webhook validation)
    cy.wait(3000);
    
    // Close the dialog regardless of success or errors
    cy.get('ui5-dialog[header-text="Upload YAML"]').within(() => {
      cy.contains('ui5-button', 'Close').should('be.visible').click();
    });
    
    cy.wait(1000);
    
    // 6. Navigate back to BTP Operator and verify counts
    cy.getLeftNav().contains('Namespaces').click();
    cy.contains('ui5-link', 'kyma-system').click();
    cy.getLeftNav().contains('BTP Operators').click();
    cy.clickGenericListLink('btpoperator');
    cy.wait(1000);
    
    // Verify Service Instances and Bindings sections are visible
    cy.getMidColumn().within(() => {
      cy.contains('Service Instances and Bindings').scrollIntoView();
      cy.wait(500);
      
      cy.contains('Service Instances').should('be.visible');
      cy.contains('Service Bindings').should('be.visible');
    });
  });
});
