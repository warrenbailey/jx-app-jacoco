#!/bin/sh
helm repo update
helm upgrade ${JX_JACOCO_ANALYZER_CHART_REPOSITORY}/ext-jacoco --version ${EXT_VERSION}
