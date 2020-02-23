#!/bin/bash

#
# Gather licenses for binary releases
#

devdeps="
gotest.tools
"

manualdeps="
github.com/snowballstem/snowball:internal/snowball/snowball/COPYING
github.com/jenil/chota:cmd/lrmon/static/LICENSE.chota.txt
"

TARGET=licenses
mkdir -p $TARGET
NOTICE=$TARGET/NOTICE
cp LICENSE $TARGET

echo "
Letarette is licensed under the Apache v2.0 License.
Letarette uses the following fine packages:
---
" > $NOTICE

while read -r dep; do
    if [ "$dep" == "" ]; then
        continue
    fi
    IFS=':' read -r -a split <<< "$dep"
    pkg="${split[0]}"
    lic="${split[1]}"
    echo $pkg >> $NOTICE
    mkdir -p "$TARGET/$pkg"
    cp "$lic" "$TARGET/$pkg"
done <<< "$manualdeps"

go mod vendor
licenses=`find vendor -name LICENSE | sed -e 's/vendor\///'`

for lic in $licenses; do
    pkg=`echo $lic | sed -e 's/\/LICENSE//'`

    for devdep in $devdeps; do
        if grep -q "^$devdep" <<< "$pkg"; then
            continue 2
        fi
    done

    if go mod why -m $pkg | grep -q "does not need"; then
        continue
    fi

    echo $pkg >> $NOTICE
    mkdir -p "$TARGET/$pkg"
    cp "vendor/$lic" "$TARGET/$pkg"
done

echo "
---

Find the license for each package in the corresponding subdirectory.
" >> $NOTICE
