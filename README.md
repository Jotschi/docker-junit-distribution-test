# dockerJVM

* [Github Project](https://github.com/Jotschi/docker-junit-distribution-test)

## Context Base Image

The docker folder contains a example for a docker base image

## Commands

```-c run```

This command will:

1. build the test context docker image and effectively stall the execution of all junit test until the image has been build and distributed.
2. Once the image has been distributed a docker container running the test will be started.

### Configuration

The *config.json*  file is used to configure the build and exec command.


Example: *config.json.example*

```
{
 "baseImageName": "hydra.sky:5000/dockersurefire",
 "contextImageName": "hydra.sky:5000/testcontext",
 "dockerswarm": "tcp://hydra.sky:2375"
}
```

* baseImageName - Image which is used during *build* command execution. This image will be used as a baseImage for the test context image.
* contextImageName - Image which will be created during *build* command execution. The image will also be pushed and used during the exec phase.
* dockerswarm - Endpoint of the docker swarm host
