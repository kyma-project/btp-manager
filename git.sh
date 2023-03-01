#!/usr/bin/env bash
ARG1=${1:-'wip - temp commit msg'}
echo "add start.."
git add  .
echo "add end"
echo "commit start.."
git commit -m "$ARG1"
echo "commit end"
echo "push start.."
git push
echo "push end"