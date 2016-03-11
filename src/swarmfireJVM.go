package main

import (
    "bytes"
    "fmt"
    "flag"
    "time"
     b64 "encoding/base64"
    "github.com/fsouza/go-dockerclient"
    "log"
    "archive/tar"
    "io/ioutil"
    "encoding/json"
    "strings"
    "os"
    "syscall"
)

const (
     LOCK_FILE = "target/.lock"
     IMAGE_FILE = "target/.imagedone"
)

func check(e error) {
  if e != nil {
     log.Fatal(e)
  }
}

func createTar() string {
  buf := new(bytes.Buffer)
  tw := tar.NewWriter(buf)
  // Add some files to the archive.
  var files = os.Args[4:]

	for _, file := range files {
    dat, err := ioutil.ReadFile(file)
    check(err)
    //fmt.Print(string(dat))
		hdr := &tar.Header{
			Name: file,
   		Mode: 0600,
  		Size: int64(len(dat)),
  	}
  	if err := tw.WriteHeader(hdr); err != nil {
     log.Fatalln(err)
  	}
  	if _, err := tw.Write([]byte(dat)); err != nil {
  		log.Fatalln(err)
  	}
  }
  // Make sure to check the error on Close.
  if err := tw.Close(); err != nil {
  	log.Fatalln(err)
  }

  sEnc := b64.StdEncoding.EncodeToString([]byte(buf.String()))
  return sEnc
}

func removeContainer(client *docker.Client, containerId string) {
  opts := docker.RemoveContainerOptions{
    ID:     containerId,
    Force:  true,
  }
  err := client.RemoveContainer(opts)
  check(err)
}

func attachTo(client *docker.Client, containerId string, outBuf *bytes.Buffer, errBuf *bytes.Buffer) {
  opts := docker.AttachToContainerOptions{
      Container: containerId,
      OutputStream: outBuf,
      ErrorStream: errBuf,
      Stdout:       true,
      //Logs:         true,
      Stderr:       true,
      Stream:       true,
  }
  fmt.Println("Attaching to container " + containerId)
  err := client.AttachToContainer(opts)
  check(err)
}

func startingContainer(client *docker.Client, containerId string) {
  fmt.Println("Starting container " + containerId)
  err2 := client.StartContainer(containerId, &docker.HostConfig{})
  check(err2)
  fmt.Println("Container executed " + containerId)
}

func createContainer(client *docker.Client, contextImageName string, fileData string) string {
  fmt.Println("Execute...")
  envs := []string{"DATA=" + fileData}
  //var cmd2 = "/opt/java/latest/bin/java -jar " +

  var cmd = []string{"/opt/java/latest/bin/java", "-jar"}
  var cmd2 = append(cmd, os.Args[4:]...)
  fmt.Println(strings.Join(cmd2 , " "))

  //cmd = []string{"find", "/home"}
  conf := docker.Config{
    Env:    envs,
    Image:  contextImageName,
    AttachStdin: false,
    AttachStdout: true,
    AttachStderr: true,
    Tty:        false,
    OpenStdin:  false,
    StdinOnce:  false,
    Cmd:    cmd2,
  }
  opts := docker.CreateContainerOptions{
//    Name:     "snapshot",
    Config:   &conf,
  }
  fmt.Println("Creating container from image " + contextImageName)
  container, err := client.CreateContainer(opts)
  check(err)
  fmt.Println("Container created..." + container.ID)
  return container.ID
}

type ClusterConfig struct {
    BaseImageName   string
    ContextImageName string
    Dockerswarm string
}

func readConfig() ClusterConfig {
  dat, err := ioutil.ReadFile("config.json")
  check(err)

  res := ClusterConfig{}
  json.Unmarshal([]byte(string(dat)), &res)
  return res
}

func writeDockerfile(baseImageName string) {
  d1 := []byte("FROM " + baseImageName + "\n")
  d1 = append(d1, "ARG BASEDIR\n"...)
  d1 = append(d1, "ENV DATA EMPTY\n"...)
  d1 = append(d1, "ADD .  ${BASEDIR}/\n"...)
  //d1 = append(d1, "RUN find /home\n"...)
  //d1 = append(d1, "ADD ./target ${BASEDIR}/target\n"...)
  d1 = append(d1, "ENTRYPOINT [\"/entrypoint.sh\"]\n"...)
  err := ioutil.WriteFile("target/Dockerfile", d1, 0644)
  check(err)
}

func pullImage(client *docker.Client, imageName string) {
  var buf bytes.Buffer
  opts := docker.PullImageOptions{
    Repository:     imageName,
    //Registry:       "",
    //Tag:            "latest",
    OutputStream: &buf,
  }
  fmt.Println("Pulling image " + imageName)
  err := client.PullImage(opts, docker.AuthConfiguration{})
  check(err)
  //fmt.Println("Out: ", buf.String())
}

func buildTestContextImage(client *docker.Client, contextImageName string) {
  var buf bytes.Buffer
  opts := docker.BuildImageOptions{
      Name:                contextImageName,
      NoCache:             true,
      //SuppressOutput:      true,
      //RmTmpContainer:      true,
      ForceRmTmpContainer: true,
      OutputStream:        &buf,
      ContextDir:          "target",
      BuildArgs:           []docker.BuildArg{{Name: "BASEDIR", Value: "/home/jotschi/workspaces/docker/docker-junit-distribution-test/target"}},
    }
    fmt.Println("Building Image " + contextImageName)
    err := client.BuildImage(opts)
    check(err)
    fmt.Println("Build Output: ", buf.String())
}

func removeImage(client *docker.Client, imageName string) {
  err := client.RemoveImage(imageName)
  check(err)
}

func pushTestContextImage(client *docker.Client, contextImageName string) {
  var buf bytes.Buffer
  opts := docker.PushImageOptions{
    Name:                 contextImageName,
    //Registry:             "hydra.sky:5000"
    OutputStream:         &buf,
  }
  err := client.PushImage(opts, docker.AuthConfiguration{})
  check(err)
  fmt.Println("Push Image: ", buf.String())

}

func build(client *docker.Client, baseImageName string, contextImageName string) {
  fmt.Println("Build...")
  pullImage(client, baseImageName)
  writeDockerfile(baseImageName)
  buildTestContextImage(client, contextImageName)
  pushTestContextImage(client, contextImageName)
}

func waitForContainer(client *docker.Client, containerId string) {
  fmt.Println("Waiting for container " + containerId)
  code, err := client.WaitContainer(containerId)
  check(err)
  fmt.Println("Container " + containerId + " terminated with code " + string(code))

}

func execute(client *docker.Client, contextImageName string) {
  var data = createTar()
  var containerId = createContainer(client, contextImageName, data)
  var outBuf bytes.Buffer
  var errBuf bytes.Buffer

  go attachTo(client, containerId, &outBuf, &errBuf)
  startingContainer(client, containerId)
  waitForContainer(client, containerId)
  log.Println("---------------------------------")

  fmt.Println(outBuf.String())
  fmt.Println(errBuf.String())
  removeContainer(client, containerId)
}

func waitForImageDone() {
  for {
    if isImageDone() {break}
    time.Sleep(1000 * time.Millisecond)
  }
}

func isImageDone() bool {
  if _, err := os.Stat(IMAGE_FILE); os.IsNotExist(err) {
    return false
  }
  return true
}

func hasLock() bool {
  if _, err := os.Stat(LOCK_FILE); os.IsNotExist(err) {
    return false
  }
  return true
}

func createImageDoneFile() {
  _, err := os.Create(IMAGE_FILE)
  if err != nil {
    fmt.Printf("%v\n", err)
  }
}

func obtainLock(conf ClusterConfig) bool {
  file, err := os.OpenFile(LOCK_FILE, os.O_CREATE+os.O_APPEND, 0666)
  if err != nil {
    fmt.Printf("%v\n", err)
  }
  fd := file.Fd()
  err = syscall.Flock(int(fd), syscall.LOCK_EX+syscall.LOCK_NB)
  if err != nil {
    return false
  } else {
    fmt.Println("Building image")
    time.Sleep(10000 * time.Millisecond)
    // Build & distribute image
    buildAndDistImage(conf)
    createImageDoneFile()
    return true
  }
}

func execTest(conf ClusterConfig) {
  client, _ := docker.NewClient(conf.Dockerswarm)
  execute(client, conf.ContextImageName)
}

func buildAndDistImage(conf ClusterConfig) {
  client, _ := docker.NewClient(conf.Dockerswarm)
  build(client, conf.BaseImageName, conf.ContextImageName);
}

func main() {
    var conf = readConfig()

    var command string
    var jarFlag string
    //flag.StringVar(&socketPath, "s", "/var/run/docker.sock", "unix socket to connect to")
    flag.StringVar(&jarFlag, "jar", "", "Surefire arguments")
    flag.StringVar(&command, "c", "help", "Command to be executed [help|run|build|tar|cleanup]")
    flag.Parse()

    switch command {
      case "build":
         buildAndDistImage(conf)
         break
      case "run":
          // If the image has not been build build the image
          // or wait until it has been build
          if ! isImageDone() {
            if hasLock() {
              fmt.Println("Waiting for image build to complete...")
              waitForImageDone()
            } else {
              if ! obtainLock(conf) {
                waitForImageDone()
              }
            }
          }
          fmt.Println("Forking test...")
          execTest(conf)
         break
      case "tar":
        createTar()
      default:
        flag.PrintDefaults()
    }
}
