#!/usr/bin/env bash
$(make debug)
$(kubectl rollout restart deployment btp-manager-controller-manager -n kyma-system)
$(clear)
$(kubectl logs -f deployment/btp-manager-controller-manager -n kyma-system)