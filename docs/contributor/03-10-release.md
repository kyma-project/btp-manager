# BTP Manager release pipeline

## Overview

The BTP Manager release pipeline creates proper artifacts:
 - btp-operator module OCI image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/btp-manager)
 - btp-manager Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/unsigned%2Fcomponent-descriptors%2Fkyma.project.io%2Fmodule%2Fbtp-operator)
 - `template.yaml`, `template_control_plane.yaml`, `btp-manager.yaml`, `btp-btp-operator-default-cr.yaml`

## Run the pipeline

### Create a release

![Release diagram](../assets/release.svg)

To create a release, follow these steps:

1. Run GitHub action **Create release**:  
   i.  go to the **Actions** tab  
   ii. click on **Create release** workflow   
   iii. click  **Run workflow** on the right  
   iv. provide a version, for example, 1.2.0  
   v. choose real or dummy credentials for Service Manager  
2. The GitHub action, defined in the [`create-release.yaml`](/.github/workflows/create-release.yaml) file, creates a GitHub tag and draft release with the provided name.
3. The GitHub action asynchronously initiates unit tests and E2E tests jobs. E2E upgrade tests run only with real credentials for Service Manager.
4. The tag creation triggers Prow Jobs, `post-btp-manager-module-build` and `post-btp-manager-build`, defined in [btp-manager-build.yaml](https://github.com/kyma-project/test-infra/blob/main/prow/jobs/btp-manager/btp-manager-build.yaml).
5. `post-btp-manager-build` builds a Docker image tagged with the release name.
6. `post-btp-manager-module-build` runs the `kyma alpha create module` command, which creates a Kyma module and pushes the image to the registry. Kyma CLI is called with the `--sec-scanners-config` flag and uses a dynamically created file to configure security scanning settings in the module template. Finally, the job uploads the `template.yaml`,`template_control_plane.yaml`, `btp-manager.yaml` and `btp-operator-default-cr.yaml` files to the btp-manager release as release assets.
7. The GitHub action waits for the `template.yaml` asset in the GitHub release and for images in the Docker registry.
8. The GitHub action fetches the module image and runs E2E tests on the k3s cluster with the specified credentials. 
9. If the unit tests and E2E tests are completed successfully, the GitHub action publishes the release.


### Replace an existing release

To regenerate an existing release, perform the following steps:

1. Delete the GitHub release.
2. Delete the GitHub tag.
3. Run the [**Create release**](#create-a-release) pipeline. 
