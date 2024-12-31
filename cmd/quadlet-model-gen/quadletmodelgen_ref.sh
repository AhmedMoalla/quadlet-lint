#!/usr/bin/env bash

set -e

generatedOutput="$PWD/$1"

workingDir=$PWD/.work
gitRoot="$workingDir/quadlet-model-gen"
rm -rf "$workingDir"
rm -rf "$generatedOutput"
mkdir "$workingDir"

[ ! -d "$gitRoot" ] && git clone --quiet -b main --single-branch https://github.com/AhmedMoalla/quadlet-lint.git "$gitRoot"

cd "$gitRoot"
go generate ./...
cp -R "$gitRoot/pkg/model/generated" "$generatedOutput"

rm -rf "$workingDir"