#!/bin/sh -e

set -e


CHANGED=$(git diff-index --name-only HEAD --)
if [[ ! -z $CHANGED ]]; then
    echo "Please commit your local changes first"
    exit 1
fi

sed -i "s/Version = \".*\"/Version = \"${1}\"/" version/version.go
sed -i "s/version: .*/version: ${1}/" helm/Chart.yaml
sed -i "s/appVersion: .*/appVersion: v${1}/" helm/Chart.yaml
operator-sdk generate openapi
operator-sdk generate k8s

cp deploy/crds/*crd.yaml helm/crds/

git add . 
git commit -m "prepare release ${1}"
git push

git tag -a v${1} -m "release ${1}"
git push origin v${1}
