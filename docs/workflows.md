### Overview

## Promote to channel

Goal of workflow is to register module by using btp manger template file. Workflow takes tag and downloads module template file from realase on github, and place it to according folder in module folder inside kyma repository, by creating pull request. The pull request need to by approved manually.

Inputs:
- Release tag: if not specified workflow will take tag from latest avaiablie realase on github and use it. Tag can also be specified directly, the workflow will validate if given tag exists on any relase and if yes, then use it, otherwise the workflow will break execution.
- Channel: there are three options as value: alpha, fast, regular. This options corresponds to folders in modules folder at Kyma repo. 