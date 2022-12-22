#! /bin/bash

export file=chart/btp-manager/values.yaml
cat $file | sed -e "s/tag.*$/tag: $1/" > tmp.yaml
mv tmp.yaml $file
