---
title: BTP Manager release pipeline
---

## Overview

The BTP Manager release pipeline creates proper artifacts:
 - btp-operator module OCI image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/btp-manager)
 - btp-manager Docker image in the [registry](http://europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator)
 - template.yaml, template_control_plane.yaml, rendered.yaml

## Run the pipeline

### Create a release
To create a release, follow these steps:

1. Run GitHub action **Create a release**: 
   1. go to the **Actions** tab
   2. click on **Create a release** workflow 
   3. click  **Run workflow** on the right
   4. provide a version, for example, v1.2.0.
2. The GitHub action, defined in the `.github/workflows/create-release.yaml` file, creates a GitHub tag and draft release with the provided name.
3. The GitHub action initiates asynchronously unit tests and E2E tests jobs.
4. The tag creation triggers Prow Jobs, `post-btp-manager-module-build` and `post-btp-manager-build`, defined in [btp-manager-build.yaml](https://github.com/kyma-project/test-infra/blob/main/prow/jobs/btp-manager/btp-manager-build.yaml).
5. `post-btp-manager-build` builds a Docker image tagged with the release name.
6. `post-btp-manager-module-build` runs the `kKyma alpha create module` command, which creates a Kyma module, and pushes the image to the registry. 
Finally, the job uploads the `template.yaml`,`template_control_plane.yaml` and `rendered.yaml` files to the btp-manager release as a release assets.
7. The GitHub action waits for the `template.yaml` asset in the GitHub release and for images in the Docker registry.
8. The GitHub action fetches module image and runs E2E tests on the k3s cluster. 
9. If unit tests and E2E tests pass GitHub action publishes the release.

```mermaid
   sequenceDiagram
      actor Gopher
      participant Github Actions
      participant Unit tests job
      participant E2E tests job
      participant Github Repository
      participant post btp manager build
      participant post btp manager module build
      participant docker Registry
      Gopher->>Github Actions: Executes
      activate Github Actions   
      Github Actions->>Github Repository: creates the tag and the draft release
      Github Actions->>Unit tests job: initiates
      activate Unit tests job
      Github Repository->>post btp manager build: triggers
      activate post btp manager build
      Note over post btp manager build: builds binary image
      Github Repository->>post btp manager module build: triggers
      deactivate Unit tests job
      Unit tests job->>Github Actions: returns result
      activate post btp manager module build
      Note over post btp manager module build: builds OCI module image and create yaml artifacts
      post btp manager build->>docker Registry: uploads docker image 
      deactivate post btp manager build
      post btp manager module build->>docker Registry: uploads OCI module image
      post btp manager module build->>Github Repository: uploads yaml artifacts
      deactivate post btp manager module build
      loop Every 10s
        E2E tests job-->docker Registry: are images available?
      end
      activate E2E tests job
      Docker registry->>E2E tests job: fetches binary image, module image
      Note over E2E tests job: create k3s cluster and run E2E tests
      E2E tests job->>Github Actions: return result
      deactivate E2E tests job
      Github Actions->>Github Repository: publish the release
      deactivate Github Actions
```

### Replace an existing release

To regenerate an existing release, perform the following steps:

1. Delete the GitHub release
2. Delete the GitHub tag
3. Run the [**Create a release**](#create-a-release) pipeline 
