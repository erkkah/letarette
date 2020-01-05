#!/bin/bash

package=${1:?Expected package argument}

if output=$(git status --porcelain) && [ -z "$output" ]; then
    # clean
    echo clean repo >&2
    sha=$(git rev-parse HEAD)
    rev=${sha:0:7}
    echo rev:$rev >&2
else
    # dirty
    echo dirty repo >&2
    echo "$output" >&2
    rev="dev"
fi

if tag=$(git tag --contains) && [ -z "$tag" ]; then
    tag=$(date +%F)
fi

echo "-X '$package.Revision=$rev' -X '$package.Tag=$tag'"
