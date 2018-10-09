#!/bin/sh
helm repo update
helm install ${JX_EXT_JACOCO_CHART_REPOSITORY}/ext-jacoco --version ${EXT_VERSION}
