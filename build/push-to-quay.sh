#!/bin/bash
echo "$QUAY_BOT_PASSWORD" | docker login -u "$QUAY_BOT_USERNAME" --password-stdin quay.io


for img in "$@"; do
  if [ -z "${TRAVIS_TAG}" ]; then
    docker ${img}:${TRAVIS_BRANCH} ${img}:${TRAVIS_TAG}
    docker ${img}:${TRAVIS_BRANCH} ${img}:latest
    docker push ${img}:${TRAVIS_TAG}
    docker push ${img}:latest
  else
    docker push ${img}:${TRAVIS_BRANCH}
  fi
done


