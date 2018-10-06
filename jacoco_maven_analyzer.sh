#!/bin/sh
jx step collect --provider=${JX_JACOCO_MAVEN_PUBLISH_PROVIDER} --pattern=target/site/jacoco/jacoco.xml --classifier=jacoco
