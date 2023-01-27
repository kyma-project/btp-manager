## BTP Manager release pipeline

The BTP Manager release pipeline creates proper artifacts:
 - btp-operator module OCI image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/btp-manager)
 - btp-manager Docker image in the [registry](http://europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator)
 - template.yaml

## Running the pipeline
How to create a release:

1. Run Github action "Create a release": go to `actions` tab, click on "Create a release" workflow. Click `run workflow` button (right side) and provide a version, for example 1.2.0.
2. The Github action (defined in `.github/workflows/create-release.yaml` file) creates a Github tag and release with provided name.
3. The tag creation triggers Prow jobs `post-btp-manager-module-build` and `post-btp-manager-build` defined in [btp-manager-build.yaml](https://github.com/kyma-project/test-infra/blob/main/prow/jobs/btp-manager/btp-manager-build.yaml).
4. `post-btp-manager-build` builds a Docker image tagged with the release name
5. `post-btp-manager-module-build` runs `kyma alpha create module` command, which creates Kyma module, pushes the image to a registry. Finally, the job uploads the `template.yaml` file to the btp-manager release as a release asset.
6. The Github action waits for `template.yaml` asset in the Github release and waits for images in the Docker registry.

![Release diagram](./assets/release.svg)