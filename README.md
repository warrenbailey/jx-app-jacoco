# jx-app-jacoco

jx-app-jacoco provides a means for transferring a JaCoCo XML code coverage report from a [Jenkins X](https://jenkins-x.github.io/jenkins-x-website/) build to a `Fact` in the `PipelineActivity` custom resource.

You must have a Jenkins X cluster to install and use the jx-app-jacoco app.
If you do not have a Jenkins X cluster and you would like to try it out, the [Jenkins X Google Cloud Tutorials](https://jenkins-x.io/getting-started/tutorials/) is a great place to start.

[TOC level=2,3,4 markdown]: # "Table of Contents"

# Table of Contents
- [Installation](#installation)
    - [Configuration](#configuration)
- [Usage](#usage)
- [Development](#development)
    - [Prerequisites](#prerequisites)
    - [Compile the code](#compile-the-code)
    - [Run the tests](#run-the-tests)
    - [Check formatting](#check-formatting)
    - [Cleanup](#cleanup)
    - [Running the app in development](#running-the-app-in-development)
        - [Prerequisites](#prerequisites)
        - [Locally](#locally)
        - [In Dev Pod](#in-dev-pod)
- [How to contribute](#how-to-contribute)

## Installation

Using the [jx command line tool](https://jenkins-x.io/getting-started/install/), run the following command:

```bash
$ jx add app jx-app-jacoco --repository http://chartmuseum.jenkins-x.io
```

NOTE: The syntax of this command is evolving and will change.

After the installation, you can view the status of jx-app-jacoco via:

```bash
$ helm status jx-app-jacoco
```

### Configuration

The following table lists the configurable parameters of the App their default values.

| Parameter                  | Description                                    | Default   |
|----------------------------|------------------------------------------------|-----------|
| logLevel                   | Log level ([trace|debug|info|warn|error])      | info      |

## Usage

The current usage of jx-app-jacoco is limited to Maven projects.
You must configure the build section of your Maven POM file for JaCoCo to generate an XML report in addition to the default jacoco.exec file.

Example Maven POM file:

```xml
<build>
  <plugins>
     <plugin>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-maven-plugin</artifactId>
     </plugin>
     <plugin>
        <groupId>org.jacoco</groupId>
        <artifactId>jacoco-maven-plugin</artifactId>
        <version>0.8.2</version>
        <executions>
           <execution>
              <id>default-prepare-agent</id>
              <goals>
                 <goal>prepare-agent</goal>
              </goals>
           </execution>
           <execution>
              <id>prepare-xml-report</id>
              <goals>
                 <goal>report</goal>
              </goals>
              <phase>verify</phase>
           </execution>
        </executions>
     </plugin>
 </plugins>
</build>
```
NOTE: We have an open issue to not have to generate the XML report in the project.

Ensure that your _Jenkinsfile_ includes the following command, so the JaCoCo XML report is stored for later retrieval by this app.

```bash
sh "jx step stash --pattern=target/site/jacoco/jacoco.xml --classifier=jacoco"
```

JaCoCo code coverage facts for each build will now be stored in a Fact custom resource.
You can retrieve a given Fact using `kubectl`:

```bash
$ kubectl get fact -o yaml jacoco-jx.coverage-<org>-<repo>-pr-<pull-request-number>-<build-number>

apiVersion: v1
items:
- apiVersion: jenkins.io/v1
  kind: Fact
  metadata:
    creationTimestamp: 2019-03-04T12:03:58Z
    generation: 1
    name: jacoco-jx.coverage-hf-bee-spring-boot-test-pr-6-1
    namespace: jx
    resourceVersion: "8407549"
    selfLink: /apis/jenkins.io/v1/namespaces/jx/facts/jacoco-jx.coverage-hf-bee-spring-boot-test-pr-6-1
    uid: 97ba797d-3e75-11e9-90f5-42010a9c0193
  spec:
    factType: jx.coverage
    measurements:
    - measurementType: count
      measurementValue: 3
      name: Instructions-Covered
    - measurementType: count
      measurementValue: 5
      name: Instructions-Missed
    - measurementType: count
      measurementValue: 8
      name: Instructions-Total
    - measurementType: count
      measurementValue: 1
      name: Lines-Covered
    - measurementType: count
      measurementValue: 2
      name: Lines-Missed
    - measurementType: count
      measurementValue: 3
      name: Lines-Total
    - measurementType: count
      measurementValue: 1
      name: Complexity-Covered
    - measurementType: count
      measurementValue: 1
      name: Complexity-Missed
    - measurementType: count
      measurementValue: 2
      name: Complexity-Total
    - measurementType: count
      measurementValue: 1
      name: Methods-Covered
    - measurementType: count
      measurementValue: 1
      name: Methods-Missed
    - measurementType: count
      measurementValue: 2
      name: Methods-Total
    - measurementType: count
      measurementValue: 1
      name: Classes-Covered
    - measurementType: count
      measurementValue: 0
      name: Classes-Missed
    - measurementType: count
      measurementValue: 1
      name: Classes-Total
    name: jacoco-jx.coverage-hf-bee-spring-boot-test-pr-6-1
    original:
      mimetype: application/xml
      tags:
      - jacoco.xml
      url: https://raw.githubusercontent.com/hf-bee/spring-boot-test/gh-pages/jenkins-x/jacoco/hf-bee/spring-boot-test/PR-6/1/target/site/jacoco/jacoco.xml
    statements: []
    subject:
      apiVersion: jenkins.io/v1
      kind: PipelineActivity
      name: hf-bee-spring-boot-test-pr-6-1
      uid: ec22d6aa-3e69-11e9-821a-42010a9c00e6
    tags:
    - jacoco
  status: {}
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

## Development

The following paragraphs describe how to build and work with the source of this application.

### Prerequisites

The project is written in [Go](https://golang.org/), so you will need a working Go installation (Go version >= 1.11.4).

The build itself is driven by GNU [Make](https://www.gnu.org/software/make/) which also needs to be installed on your system.

### Compile the code

```bash
$ make `uname | tr '[:upper:]' '[:lower:]'`
```

After successful compilation the `jx-app-jacoco` binary can be found in the `bin` directory.

### Run the tests

```bash   
$ make test
```

### Check formatting

```bash   
$ make check
```

### Cleanup

```bash   
$ make clean
```

### Running the app in development

#### Prerequisites

* Setup a Jenkins X environment
  * [Download jx](https://jenkins-x.io/getting-started/install/)
  * [Setup cluster](https://jenkins-x.io/getting-started/create-cluster/),
    eg:

    ```bash
    $ jx create cluster gke --prow
    ```

  * Follow the instructions to complete the cluster setup

#### Locally

You can run the compiled binary locally for easy development.
To do so, you need to export the required configuration options in your shell.

```
$ make run
```

#### In Dev Pod

* Open a [Dev Pod](https://jenkins-x.io/developing/devpods/)
* In Dev Pod

  ```
  # Run once
  $ helm install --name jx-app-jacoco --set image.repository=$DOCKER_REGISTRY/jenkinsxio/jx-app-jacoco charts/jx-app-jacoco/
    
   # Run after successive changes
   $ make skaffold-run
    
   # To delete
   $ helm delete --purge jx-app-jacoco
  ```

__TIP__: If you get an error of the form `Error: pods is forbidden: User "system:serviceaccount:jx:knative-build-bot" cannot list pods in the namespace "kube-system”`, run the following patch command:

```bash
$ kubectl patch clusterrole/knative-build-bot --type 'json' -p '[{"path": "/rules/2/verbs/1", "value": "list", "op": "add"}]'
```

## How to contribute

If you want to contribute, make sure to follow the [contribution guidelines](./CONTRIBUTING.md) when you open issues or submit pull requests.
