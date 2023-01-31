---
title: Promote BTP Manager to release channel
---

The goal of the workflow is to register a module using the BTP Manager template file. The workflow takes a tag and downloads the module template file from the release on GitHub, and places it in the relevant subfolder in the module folder in the Kyma repository by creating a pull request. The pull request needs to be approved manually.

If you want to control the workflow, you can set the following inputs in the GitHub UI:
- Release tag - If not specified, the workflow takes a tag from the latest available release on GitHub and uses it. The tag can also be specified directly. The workflow validates if a given tag exists on any release and if it does, then uses it. Otherwise, the workflow breaks the execution.
- Channel - There are three release channels to choose from: beta, fast, and regular. The options correspond to the subfolders in the module folder at the Kyma repository. 


---
title: Auto update chart/resources
---

The goal of workflow is to to update chart (module-chart/chart) to newest version, render resource templates from newest chart and create PR with changes. Workflow work on two shell scripts:

- make-module-chart.sh - it download charts from https://github.com/SAP/sap-btp-service-operator from realase tagged as 'latest' and place it in module-chart/chart. 
	
- make-module-resources - it use helm to render kuberneter resources templates, as a base it takes chart from module-chart/chart and also takes values to override https://github.com/kyma-project/btp-manager/blob/main/module-chart/overrides.yaml. After helm templating finishes templating with appiled overrides, the generated resources are put into module-resources/apply, the resources which were in previous version but they are not in new version are placed under module-resource/delete
 
Both of above scripts are executed from the workflow but also can be triggered manually from developers computer. They are placed under hack/update/.