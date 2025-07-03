# BTP Manager Promote Module to Channel

## Overview

The BTP Manager Promote Module to Channel Pipeline creates PR in the `module-manifests` modifying the `module-releases.yaml` file accordingly.

## Run the Pipeline

### Promote to channel

To promote a released version, follow these steps:

1. Run GitHub action **Promote module to channel**:  
   i.  Go to the **Actions** tab  
   ii. Click **Promote module to channel** workflow   
   iii. Click  **Run workflow** on the right  
   iv. Provide a version, for example, 1.2.0  
   v. Choose regular or fast channel  
2. The GitHub action, defined in the [`promote_module_to_channel`](/.github/workflows/promote_module_to_channel.yaml) file, validates the release by checking if the GitHub tag already exists.
3. The GitHub actions creates a PR in the `module-manifests` repository with the `module-releases.yaml` file modified in the section for the specified channel.
4. A code owner approves the PR.
5. Once PR is merged Submission Pipeline is triggered that generates a ModuleReleaseMeta CR, and pushes it to the /kyma/kyma-modules repository.

