#!/usr/bin/env bash

x=$(curl -L \
            -X POST \
            -H "Accept: application/vnd.github+json" \
            -H "Authorization: Bearer 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" \
            -H "X-GitHub-Api-Version: 2022-11-28" \
            https://api.github.com/repos/kyma-project/btp-manager/issues/372/comments \
            -d '{"body":"Me too"}')

echo $x