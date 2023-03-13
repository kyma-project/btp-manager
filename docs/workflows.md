---
title: GitHub Action workflows
---

## Promote BTP Manager to release channel

The goal of the workflow is to register a module using the BTP Manager template file. The workflow takes a tag and downloads the module template file from the release on GitHub, and places it in the relevant subfolder in the module folder in the Kyma repository by creating a pull request. The pull request needs to be approved manually.

If you want to control the workflow, you can set the following inputs in the GitHub UI:
- Release tag - If not specified, the workflow takes a tag from the latest available release on GitHub and uses it. The tag can also be specified directly. The workflow validates if a given tag exists on any release and if it does, then uses it. Otherwise, the workflow breaks the execution.
- Channel - There are three release channels to choose from: beta, fast, and regular. The options correspond to the subfolders in the module folder at the Kyma repository. 

## Auto update chart and resources

The goal of the workflow is to update the chart (module-chart/chart) to the newest version, render resource templates from the newest chart, and create a PR with the changes. The workflow works on two shell scripts:

- `make-module-chart.sh` - downloads the chart from the [upstream](https://github.com/SAP/sap-btp-service-operator) from the release tagged as 'latest' and places it in the `module-chart/chart`. 
	
- `make-module-resources.sh` - uses Helm to render Kubernetes resources templates. As a base it takes a the chart from the `module-chart/chart` and also takes values to [override](https://github.com/kyma-project/btp-manager/blob/main/module-chart/overrides.yaml). After Helm finishes templating with the applied overrides, the generated resources are put into `module-resources/apply`. The resources that were used in the previous version but are not used in the new version are placed under `module-resource/delete`.
During the process of iterating over the `sap-btp-service-operator` resources, the script also keeps track of the GVKs to generate RBAC rules in [`btpoperator\_controller.go`](https://github.com/kyma-project/btp-manager/blob/5a8420347c6a526f158fde7c41c3842eb54e2fda/controllers/btpoperator_controller.go#L135-L146) which feeds into RBAC `ClusterRole` in [`role.yaml`](https://github.com/kyma-project/btp-manager/blob/5a8420347c6a526f158fde7c41c3842eb54e2fda/config/rbac/role.yaml#L1) resource
kept in sync with `make manifests` just like any standard [kubebuilder operator](https://book-v2.book.kubebuilder.io/reference/markers/rbac.html). The script excludes all resources with a Helm hook "pre-delete" which is not necessary in resources to apply. Additionally all excluded resources are added to `module-resources/excluded` folder to see, what was excluded.
 
Both scripts are run from the workflow but can also be triggered manually from the developer's computer. They are placed under `hack/update/`.

To keep `module-chart/chart` in sync with the [upstream](https://github.com/SAP/sap-btp-service-operator), you must not apply any manual changes there.
