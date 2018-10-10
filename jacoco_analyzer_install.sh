#!/bin/sh
helm repo update
helm install ${JX_JACOCO_ANALYZER_INSTALL_CHART_REPOSITORY}/ext-jacoco --version ${EXT_VERSION} --set teamNamespace=${EXT_TEAM_NAMESPACE} --name=ext-jacoco
