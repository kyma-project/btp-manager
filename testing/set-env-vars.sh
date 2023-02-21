#!/bin/bash

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION=1.25.0

LOCALBIN=`pwd`/bin
ENVTEST=${LOCALBIN}/setup-envtest

KUBEBUILDER_ASSETS="$(${ENVTEST} use ${ENVTEST_K8S_VERSION} --bin-dir ${LOCALBIN} -p path)"

# not existing cluster by default
USE_EXISTING_CLUSTER=${USE_EXISTING_CLUSTER:="false"}

# if you plan to debug or run on existing cluster increase the timeout (30 seconds should be ok)
SINGLE_TEST_TIMEOUT=${SINGLE_TEST_TIMEOUT:="5s"}

# if you plan to debug or run on existing cluster increase the timeout (180 seconds should be ok)
SUITE_TIMEOUT=30s

# setting verbose output for ginkgo
# for very verbose set "ginkgo.vv"
GINKGO_VERBOSE_FLAG=${GINKGO_VERBOSE_FLAG:="ginkgo.v"}

# GINKGO_LABEL_FILTER="provisioning,test-update"
GINKGO_LABEL_FILTER=${GINKGO_LABEL_FILTER:=""}

# should be false for env-test cluster, may be true for existing cluster
DISABLE_WEBHOOK_FILTER_FOR_TESTS=${DISABLE_WEBHOOK_FILTER_FOR_TESTS:="false"}

export KUBEBUILDER_ASSETS
export USE_EXISTING_CLUSTER
export SINGLE_TEST_TIMEOUT
export SUITE_TIMEOUT
export GINKGO_LABEL_FILTER
export GINKGO_VERBOSE_FLAG
export DISABLE_WEBHOOK_FILTER_FOR_TESTS
