#!/bin/sh
helm repo update
helm install chartmuseum/ext-jacoco --version ${EXT_VERSION}
