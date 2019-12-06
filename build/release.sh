#!/bin/sh -e

set -e

CHANGED=$(git diff-index --name-only HEAD --)
if [[ ! -z $CHANGED ]]; then
    echo "Please commit your local changes first"
    exit 1
fi

RELEASE=${1}

sed -i "s/Version = \".*\"/Version = \"${RELEASE}\"/" version/version.go
sed -i "s/version: .*/version: ${RELEASE}/" helm/Chart.yaml
sed -i "s/appVersion: .*/appVersion: v${RELEASE}/" helm/Chart.yaml
operator-sdk generate openapi
operator-sdk generate k8s

cp deploy/crds/*crd.yaml helm/crds/

git add . 
git commit -m "prepare release ${RELEASE}"
git push

git tag -a v${RELEASE} -m "release ${RELEASE}"
git push origin v${RELEASE}

curl --header "Content-Type: application/json" \
  --header "Authorization: token ${GITHUB_TOKEN}}" \
  --request POST \
  --data "{
  \"tag_name\": \"v${RELEASE}\",
  \"name\": \"v${RELEASE}\",
  \"body\": \"Release v${RELEASE}\",
  \"draft\": false,
  \"prerelease\": false
}" https://api.github.com/repos/bakito/k8s-event-logger-operator/releases