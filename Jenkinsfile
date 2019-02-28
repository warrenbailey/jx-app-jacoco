pipeline {
    agent any
    environment {
      ORG               = 'jenkinsxio'
      GITHUB_ORG        = 'jenkins-x-apps'
      APP_NAME          = 'jx-app-jacoco'
      GIT_PROVIDER      = 'github.com'
      CHARTMUSEUM_CREDS = credentials('jenkins-x-chartmuseum')
    }
    stages {
      stage('CI Build and push snapshot') {
        when {
          branch 'PR-*'
        }
        environment {
          PREVIEW_VERSION = "0.0.0-SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER"
        }
        steps {
          dir ('/home/jenkins/go/src/github.com/jenkins-x-apps/jx-app-jacoco') {
            checkout scm
            sh "make linux test check"
            sh 'export VERSION=$PREVIEW_VERSION && make skaffold-build'
          }
        }
      }
      stage('Build Release') {
        when {
          branch 'master'
        }
        steps {
          dir ('/home/jenkins/go/src/github.com/jenkins-x-apps/jx-app-jacoco') {
            git 'https://github.com/jenkins-x-apps/jx-app-jacoco'
            sh "git checkout master"
            sh "git config --global credential.helper store"
            sh "jx step git credentials"
            sh "echo \$(jx-release-version) > VERSION"
            sh "make release"
          }
        }
      }
    }
  }
