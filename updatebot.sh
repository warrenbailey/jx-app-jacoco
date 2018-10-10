#!/bin/sh
updatebot push-regex -r "\s+tag: (.*)" -v $(cat VERSION) --previous-line "\s+remote: github.com/${ORG}/${APP_NAME}" jenkins-x-extension-definitions.yaml
