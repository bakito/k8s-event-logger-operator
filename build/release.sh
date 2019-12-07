#!/bin/sh -e

set -e

CHANGED=$(git diff-index --name-only HEAD --)
if [[ ! -z $CHANGED ]]; then
    echo "Please commit your local changes first"
    exit 1
fi

RELEASE=${1}

sed -i "s/Version = \".*\"/Version = \"v${RELEASE}\"/" version/version.go
sed -i "s/version: .*/version: ${RELEASE}/" helm/Chart.yaml
sed -i "s/appVersion: .*/appVersion: v${RELEASE}/" helm/Chart.yaml
operator-sdk generate openapi
operator-sdk generate k8s

cp deploy/crds/*crd.yaml helm/crds/

git add . 
git commit -m "prepare release ${RELEASE}"
git push

RELEASE_ID=$(curl --header "Content-Type: application/json" \
  --header "Authorization: token ${GITHUB_TOKEN}" \
  --request POST \
  --data "{
  \"tag_name\": \"v${RELEASE}\",
  \"name\": \"v${RELEASE}\",
  \"body\": \"Release v${RELEASE}\n\nHelm Chart: [k8s-event-logger-operator-${RELEASE}.tgz](https://github.com/bakito/k8s-event-logger-operator/releases/download/v${RELEASE}/k8s-event-logger-operator-${RELEASE}.tgz) \",
  \"draft\": false,
  \"prerelease\": false
}" https://api.github.com/repos/bakito/k8s-event-logger-operator/releases)

helm package ./helm/ --version ${RELEASE} --app-version v${RELEASE}

curl \
  -H "Authorization: token $GITHUB_TOKEN" \
  -H "Content-Type: $(file -b --mime-type ${FILE})" \
  --data-binary @${FILE} \
  "https://uploads.github.com/repos/bakito/k8s-event-logger-operator/releases/${RELEASE_ID}/assets?name=$(basename ${FILE})"

rm -Rf ${FILE}