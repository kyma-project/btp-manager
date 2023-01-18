---
title: Run unit tests
---
##Run unit tests using CLI 

To run the unit tests from the command line, use the following command from the BTP Manager main directory: 

```shell
make test
```
The details concerning the `test` rule (prerequisites and recipe) are defined in `./Makefile`.

By default, the unit tests are run using the envtest cluster. 
Some of the unit tests are implemented using the [Gingko](https://onsi.github.io/ginkgo/#top) library, but all the unit tests are invoked using the `go test ./... <some options>`.
You can find the exact invocation reflected in the console output along with messages confirming whether the envtest is used.
```
STEP: bootstrapping test environment @ 01/13/23 08:24:45.981
2023-01-13T08:24:45.981+0100  DEBUG   controller-runtime.test-env     starting control plane
```

### Run unit tests on existing cluster

You can run the tests on an existing cluster (not the envtest) setting the value of the environment variable **USE_EXISTING_CLUSTER** to `true`.

```shell
USE_EXISTING_CLUSTER=true make test
```

> **NOTE:** The test suite assumes the proper state of the cluster before running. If problems with left-over resources occur, you can recreate the cluster or remove resources manually.

### Test output verbosity

Setting for the `go test` verbosity is `-v` (verbose, print the full output event for passing tests). This can be changed in `make` recipe. 
For `ginkgo` tests execution default setting is `-ginkgo.v` (verbose) and can be changed e.g. for `very verbose` by setting environment variable `GINKGO_VERBOSE_FLAG`.
Allowed values are: `ginkgo.succinct`, `ginkgo.v`, `ginkgo.vv`. Accordingly, the output level will be: succinct, verbose, very verbose.

```shell
GINKGO_VERBOSE_FLAG="ginkgo.vv" make test
```

### Filtering labels
You can use `gingko` library labeling features to filter which tests specs are to be executed 
(for details see [Spec Labels](https://onsi.github.io/ginkgo/#spec-labels) in ginkgo documentation). In order to use labels for filtering 
you need to instrument the test nodes (`Describe`, `It`, `When` et al.) in `./controllers/btpoperator_controller_test.go` with labels e.g.:
```go
	Describe("Provisioning", Label("test-provisioning", "smoke-test"), func() {
```
```go
	Describe("Deprovisioning", Label("test-deprovisioning"), func() {
```

You can use labels either by setting `GINKGO_LABEL_FILTER` variable. For example to run only specs labeled as `smoke-test`:
```shell
GINKGO_VERBOSE_FLAG="smoke-test" make test
```

Another example of simple expression:
```shell
GINKGO_VERBOSE_FLAG="test-provisioning,test-deprovisioning" make test
```

### Environment variables

All above-mentioned environment variables can be also set in `testing/set-env-vars.sh`. The script sets default values for all environment variables used in `go test` invocation. 
Changing script contents is recommended when more complex filtering expression is required, or you frequently reuse the setting. However, you should not push the changes without considering 
how this affects Github Actions workflows.

### Running test suite with IDE

You can define environment variables in the Run Configuration and in effect run tests on existing cluster changing logs verbosity and using filtering features.

<img src="./assets/test-run-configuration.png" width="50%" height="50%">


<img src="./assets/environment-variable-for-test-run-config.png" width="50%" height="50%0">
