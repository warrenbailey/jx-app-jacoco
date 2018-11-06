#!/bin/sh
NAME=ext-jacoco
helm install --repo ${JX_JACOCO_ANALYZER_INSTALL_CHART_REPOSITORY} ${NAME} --version ${EXT_VERSION} --set teamNamespace=${EXT_TEAM_NAMESPACE} --name=${NAME}
