#!/bin/sh

updatebot push-regex -r "\s+tag: (.*)" -v v$(cat VERSION) --previous-line "\s+-?\s+remote: ${GIT_PROVIDER}/${GITHUB_ORG}/${APP_NAME}" ./scripts/jenkins-x-extension-definitions.yaml

