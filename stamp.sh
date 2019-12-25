package=${1:?Expected package argument}

if output=$(git status --porcelain) && [ -z "$output" ]; then
    # clean
    sha=$(git rev-parse HEAD)
    rev=${sha:0:7}
else
    # dirty
    rev="dev"
fi

if tag=$(git tag) && [ -z "$tag"]; then
    tag=$(date +%F)
fi

echo "-X '$package.Revision=$rev' -X '$package.Tag=$tag'"
