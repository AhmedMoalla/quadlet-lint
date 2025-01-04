#!/usr/bin/env bash

set -e

generatedOutput="$PWD/$1"

gitRoot="/tmp/quadlet-model-gen-$(((RANDOM<<15)|RANDOM))"
rm -rf "$generatedOutput"

currentGitRoot=$(git rev-parse --show-toplevel)
cp -R "$currentGitRoot" "$gitRoot"

cd "$gitRoot"
git reset --quiet --hard
git switch main --quiet
git reset --quiet --hard origin/main
go generate ./...
cp -R "$gitRoot/pkg/model/generated" "$generatedOutput"

rm -rf "$gitRoot"