#!/bin/sh
mvn org.jacoco:jacoco-maven-plugin:${JX_JACOCO_MAVEN_VERSION}:prepare-agent test org.jacoco:jacoco-maven-plugin:${JX_JACOCO_MAVEN_VERSION}:report
