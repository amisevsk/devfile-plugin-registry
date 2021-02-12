#!/bin/bash

readarray -d '' metas < <(find /usr/local/apache2/htdocs/v3 -name 'meta.yaml' -print0)
for meta in ${metas[@]}; do
  outputDir="${meta%/meta.yaml}"
  outputDir="${outputDir#/usr/local/apache2/htdocs/}"
  outputDir="/build/$outputDir"
  echo "Converting $meta to ${outputDir}/devfile.yaml"
  mkdir -p "$outputDir"
  /plugin-convert --from $meta --to "$outputDir/devfile.yaml"
done