#! /bin/bash

export file=chart/btp-manager/Chart.yaml
cat $file | sed -e "s/version.*$/version: $1/" > tmp.yaml
mv tmp.yaml $file
