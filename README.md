# jx-app-jacoco

jx-app-jacoco provides a means for transferring a Jacoco XML code coverage report from a [Jenkins X](https://jenkins-x.github.io/jenkins-x-website/) build to a `Fact` in the `PipelineActivity` custom resource.

You must have a Jenkins X cluster to install and use the jx-app-jacoco app.
If you do not have a Jenkins X cluster and you would like to try it out, the [Jenkins X Google Cloud Tutorials](https://jenkins-x.io/getting-started/tutorials/) is a great place to start.

## Installation

Using the [jx command line tool](https://jenkins-x.io/getting-started/install/), run the following command:

```bash
$ jx add app jx-app-jacoco --repository "http://chartmuseum.jenkins-x.io"
```

NOTE: The syntax of this command is evolving and will change.

Upon successful installation, you should see jx-app-jacoco in the list of pods (`kubectl get pods`) running in your cluster - it will be called `jx-app-jacoco-jx-app-jacoco`.
                                                                                                        
NOTE: The name repetition is a typical pattern in Helm.

## Usage

The current usage of jx-app-jacoco is limited to Maven projects.
You must configure the build section of your Maven POM file for Jacoco to generate an XML report in addition to the default jacoco.exec file.

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

Ensure that your _Jenkinsfile_ includes the following command, so the Jacoco XML report is stored for later retrieval by this app.

```bash
sh "jx step stash --pattern=target/site/jacoco/jacoco.xml --classifier=jacoco"
```

Example JenkinsFile build steps:

```bash
sh "mvn install"
sh "jx step stash --pattern=target/site/jacoco/jacoco.xml --classifier=jacoco"
```

Jacoco code coverage facts will now be stored in the PipelineActivity custom resource for each build.

```
$ kubectl get act -o yaml <org>-<repo>-pr-<pull request number>-<build-number>
```

```yaml
factType: jx.coverage
id: 0
measurements:
- measurementType: percent
  measurementValue: 6
  name: Instructions-Coverage
- measurementType: percent
  measurementValue: 7
  name: Instructions-Missed
- measurementType: percent
  measurementValue: 13
  name: Instructions-Total
- measurementType: percent
  measurementValue: 2
  name: Lines-Coverage
- measurementType: percent
  measurementValue: 3
  name: Lines-Missed
- measurementType: percent
  measurementValue: 5
  name: Lines-Total
- measurementType: percent
  measurementValue: 2
  name: Complexity-Coverage
- measurementType: percent
  measurementValue: 2
  name: Complexity-Missed
- measurementType: percent
  measurementValue: 4
  name: Complexity-Total
- measurementType: percent
  measurementValue: 2
  name: Methods-Coverage
- measurementType: percent
  measurementValue: 2
  name: Methods-Missed
- measurementType: percent
  measurementValue: 4
  name: Methods-Total
- measurementType: percent
  measurementValue: 2
  name: Classes-Coverage
- measurementType: percent
  measurementValue: 0
  name: Classes-Missed
- measurementType: percent
  measurementValue: 2
  name: Classes-Total
```

## Building from source

The following paragraphs describe how to build and work with the source.

### Prerequisites

The project is written in [Go](https://golang.org/), so you will need a working Go installation (Go version >= 1.11.4).

The build itself is driven by GNU [Make](https://www.gnu.org/software/make/) which also needs to be installed on your system.

### Compile the code

```bash
$ make linux
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

### Testing a development version of the app

* Install a JenkinsX dev cluster - see [`jx cluster create`](https://jenkins-x.io/getting-started/install/)
* Install the latest jacoco app 
   ```bash
   $ jx add app jx-app-jacoco --repository "http://chartmuseum.jenkins-x.io"
   ```
* Start a synced DevPod from your checked out sources - [Using Jenkins X DevPods](https://jenkins.io/blog/2018/06/21/jenkins-x-devpods/)
* In the DevPod
   ```bash
   $ make skaffold-build VERSION=<your-dev-version>
   ```
* Locally patch the currently deployed image
   ```bash
   $ export VERSION=<your-dev-version>
   $ export DOCKER_REGISTRY=`kubectl get service jenkins-x-docker-registry -o go-template --template="{{.spec.clusterIP}}"`:5000
   $ kubectl patch deployment jx-app-jacoco-jx-app-jacoco --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value":"'$DOCKER_REGISTRY'/jenkins-x-apps/jx-app-jacoco:'$VERSION'"}]'
   ```

## How to contribute

If you want to contribute, make sure to follow the [contribution guidelines](./CONTRIBUTING.md) when you open issues or submit pull requests.
