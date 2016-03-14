# swarmfire

Maven surefire docker bridge

[![Build Status](https://secure.travis-ci.org/jotschi/swarmfire.png)](http://travis-ci.org/jotschi/swarmfire)


* TL,DR
   * Swarmfire is a small tool written in go which creates a bridge between the [maven-surefire-plugin](https://maven.apache.org/surefire/maven-surefire-plugin/) and [docker swarm](https://docs.docker.com/swarm/) in order to execute distributed junit tests across multiple hosts.
   * A new docker container is started which runs the JVM instead of forking a JVM on the host. This way junit tests can be distributed across multiple hosts.

It is possible to execute junit tests in a multithreaded fashion using the *forkcount* parameter. Additionally the JVM executable can also be specified.
Normally a new JVM will be forked if the reuseFork parameter is set to false and threadCount is set to 1.
Instead of using the host JVM a docker container will be spawned which will execute the test. The swarmfire tool will spawn a new docker container and also apply a bit of magic.

Swarmfire will create new docker image (testcontext image) which includes all build dependencies that are needed to execute the junit test. In order to add all the needed maven dependencies the initial maven build must be triggered using the ```-Dmaven.repo.local=target/.m2``` option. This way the .m2 local repository will be placed in your target folder and thus be included in the testcontext image.

## Example Project

* [Github Project](https://github.com/Jotschi/swarmfire-example)

## Base Image

The testcontext image is based upon the [swarmfire base image](https://hub.docker.com/r/jotschi/swarmfire/)

## Commands

```-c run```

This command will build the test context docker image and effectively stall the execution of all junit test until the image has been build.
When ready a new docker container will be created and the test will be invoked within the container.

### Configuration

The *config.json*  file is used to configure the build and exec command.

Example: *config.json.example*

```
{
 "baseImageName": "jotschi/swarmfire",
 "contextImageName": "hydra.sky:5000/testcontext",
 "dockerswarm": "tcp://hydra.sky:2375",
 "command": ["java", "-jar"]
}
```

* baseImageName - Image which is used during *build* command execution. This image will be used as a baseImage for the test context image. Default: jotschi/swarmfire
* contextImageName - Image which will be created during *build* command execution. The image will also be pushed and used during the exec phase.
* dockerswarm - Endpoint of the docker swarm host
* command - Command that will be used within the docker container to start the java process

## Limitations / Issues

* I have only tested swarmfire using junit other test providers may not work.
* Currently no image cleanup is implemented. The test context images will remain on the docker hosts and the docker repository.
* Only tested using maven surefire plugin 2.19.1
