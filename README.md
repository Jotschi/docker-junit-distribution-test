# swarmfire

Swarmfire

* TL,DR
  * A new docker container is started which runs the JVM instead of forking a JVM on the host. This way junit tests can be distributed across multiple hosts.

It is possible to execute junit tests in a multithreaded fashion using the forkcount parameter. Additionally the JVM can also be specified.
Normally a new JVM will be forked when the reuseFork parameter is set to false and threadCount is set to 1.
Instead of the host JVM a docker container will be spawned which will execute the test. The dockerJVM tool will spawn a new docker container.

## Workflow

* Run mvn with -Dmaven.repo.local=target/.m2
* Invocation of dockerJVM -c build using the exec plugin in order to create a new test context image which contains .m2 files and classes
* Invocation of surfire:test using dockerJVM for JVM parameter. This way a new docker container will be created and started

## swarmfireJVM

[SwarmfireJVM](https://github.com/Jotschi/swarmfire/tree/swarmfireJVM)

* [Github Project](https://github.com/Jotschi/swarmfire/tree/swarmfire)
* [Base Image](https://hub.docker.com/r/jotschi/swarmfire/)

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
