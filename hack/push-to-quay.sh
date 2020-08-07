#!/bin/bash
set -e
echo "$QUAY_BOT_PASSWORD" | docker login -u "$QUAY_BOT_USERNAME" --password-stdin quay.io

for img in "$@"; do
  if [ -z "${TRAVIS_TAG}" ]; then
    docker push ${img}:${TRAVIS_BRANCH}
  else
    docker tag ${img}:${TRAVIS_BRANCH} ${img}:${TRAVIS_TAG}
    docker tag ${img}:${TRAVIS_BRANCH} ${img}:latest
    docker push ${img}:${TRAVIS_TAG}
    docker push ${img}:latest
  fi
done

docker logout quay.io
