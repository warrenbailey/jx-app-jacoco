#!/bin/sh
helm repo update
helm upgrade ext-jacoco ${JX_JACOCO_ANALYZER_UPGRADE_CHART_REPOSITORY}/ext-jacoco --version ${EXT_VERSION} --set teamNamespace=${EXT_TEAM_NAMESPACE}
