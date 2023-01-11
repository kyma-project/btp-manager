## Promote BTP Manager to a release channel

### Overview

The goal of the workflow is to register a module using the BTP Manager template file. The workflow takes a tag and downloads the module template file from the release on GitHub, and places it in the relevant subfolder in the module folder in the Kyma repository by creating a pull request. The pull request needs to be approved manually.

If you want to control the workflow, you can set the following inputs in the GitHub UI:
- Release tag - If not specified, the workflow takes a tag from the latest available release on GitHub and uses it. The tag can also be specified directly. The workflow validates if a given tag exists on any release and if it does, then uses it. Otherwise, the workflow breaks the execution.
- Channel - There are three release channels to choose from: alpha, fast, and regular. The options correspond to the subfolders in the module folder at the Kyma repository. 