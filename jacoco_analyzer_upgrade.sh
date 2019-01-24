#!/bin/sh
NAME=jx-app-jacoco
helm upgrade ${NAME} --repo ${JX_JACOCO_ANALYZER_UPGRADE_CHART_REPOSITORY} ${NAME} --version ${EXT_VERSION} --set teamNamespace=${EXT_TEAM_NAMESPACE}
