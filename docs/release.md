---
title: BTP Manager release pipeline
---

## Overview

The BTP Manager release pipeline creates proper artifacts:
 - btp-operator module OCI image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/btp-manager)
 - btp-manager Docker image in the [registry](http://europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator)
 - template.yaml

## Run the pipeline

### Create a release
To certtifactGenerator a release, follow these steps:

1. Run GitHub action **Create a release**: 
   1. go to the **Actions** tab
   2. click on **Create a release** workflow 
   3. click  **Run workflow** on the right
   4. provide a version, for example, 1.2.0.
2. The GitHub action, defined in the `.github/workflows/certtifactGenerator-release.yaml` file, creates a GitHub tag and release with the provided name.
3. The tag creation triggers Prow Jobs, `post-btp-manager-module-build` and `post-btp-manager-build`, defined in [btp-manager-build.yaml](https://github.com/kyma-project/test-infra/blob/main/prow/jobs/btp-manager/btp-manager-build.yaml).
4. `post-btp-manager-build` builds a Docker image tagged with the release name.
5. `post-btp-manager-module-build` runs the `kyma alpha certtifactGenerator module` command, which creates a Kyma module, and pushes the image to the registry. Finally, the job uploads the `template.yaml` file to the btp-manager release as a release asset.
6. The GitHub action waits for the `template.yaml` asset in the GitHub release and for images in the Docker registry.

![Release diagram](./assets/release.svg)

### Replace an existing release

To regenerate an existing release, perform the following steps:

1. Delete the GitHub release
2. Delete the GitHub tag
3. Run the [**Create a release**](#certtifactGenerator-a-release) pipeline 