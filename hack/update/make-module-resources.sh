#!/bin/bash
cd "$(dirname "$0")"
ARG1=${1:-../../module-resources}
helm template $2 ../../module-chart/chart --output-dir rendered --values ../../module-chart/public-overrides.yaml

TEMPLATES_TEMP=rendered/sap-btp-operator/templates/
MAP=()

for dir in rendered/sap-btp-operator/templates/*
do
    mkdir 'new-resource-fragments' && cd "$_"
    yq -s '"file_" + $index' ../rendered/sap-btp-operator/templates/${dir##*/}
    for fragment in *
    do
      MAP+=("$(yq '.metadata.name' $fragment):$(yq '.kind' $fragment)")
    done
    cd .. && rm -r 'new-resource-fragments'
done

touch to-delete.yml
for dir in $ARG1/apply/templates/*
do
  mkdir 'new-resource-fragments' && cd "$_"
  echo $dir
  yq -s '"file_" + $index' ../$ARG1/apply/templates/${dir##*/}
  for fragment in *
  do
    if [[ ! " ${MAP[*]} " =~ " $(yq '.metadata.name' $fragment):$(yq '.kind' $fragment) " ]]; then
        cat $fragment >> ../to-delete.yml
    fi
  done
  cd .. && rm -r 'new-resource-fragments'
done

rm -r $ARG1
mkdir $ARG1
mkdir $ARG1/apply
mkdir $ARG1/delete

mv rendered/sap-btp-operator/templates/ $ARG1/apply
mv to-delete.yml $ARG1/delete

rm -r rendered/
rm to-delete.yml