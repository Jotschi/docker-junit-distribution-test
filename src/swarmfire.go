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
  "io"
  "syscall"
)

const (
  LOCK_FILE = "target/.lock"
  IMAGE_FILE = "target/.imagedone"
)

func check(e error) {
  if e != nil {
    log.Fatal(e)
    os.Exit(1)
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

func createContainer(client *docker.Client, conf ClusterConfig, fileData string) string {
  fmt.Println("Execute...")
  envs := []string{"DATA=" + fileData}
  //var cmd2 = "/opt/java/latest/bin/java -jar " +

  var cmd = conf.Command
  var cmd2 = append(cmd, os.Args[4:]...)
  fmt.Println(strings.Join(cmd2 , " "))

  dconf := docker.Config{
    Env:    envs,
    Image:  conf.ContextImageName,
    AttachStdin: false,
    AttachStdout: true,
    AttachStderr: true,
    Tty:        false,
    OpenStdin:  false,
    StdinOnce:  false,
    Cmd:    cmd2,
  }
  opts := docker.CreateContainerOptions{
    Config:   &dconf,
  }
  fmt.Println("Creating container from image " + conf.ContextImageName)
  container, err := client.CreateContainer(opts)
  check(err)
  fmt.Println("Container created..." + container.ID)
  return container.ID
}

type ClusterConfig struct {
  BaseImageName   string
  ContextImageName string
  Dockerswarm string
  Command []string
}

func readConfig() ClusterConfig {
  dat, err := ioutil.ReadFile("swarmfire-config.json")
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
  d1 = append(d1, "ENTRYPOINT [\"/entrypoint.sh\"]\n"...)
  err := ioutil.WriteFile("target/Dockerfile", d1, 0644)
  check(err)
}

func pullImage(client *docker.Client, imageName string) {
  var buf bytes.Buffer
  opts := docker.PullImageOptions{
    Repository:     imageName,
    OutputStream: &buf,
  }
  fmt.Println("Pulling image " + imageName)
  err := client.PullImage(opts, docker.AuthConfiguration{})
  check(err)
}

func buildTestContextImage(client *docker.Client, contextImageName string) {
  var buf bytes.Buffer

  pwd, err := os.Getwd()
  check(err)

  opts := docker.BuildImageOptions{
    Name:                contextImageName,
    NoCache:             true,
    //SuppressOutput:      true,
    //RmTmpContainer:      true,
    ForceRmTmpContainer: true,
    OutputStream:        &buf,
    ContextDir:          "target",
    BuildArgs:           []docker.BuildArg{{Name: "BASEDIR", Value: pwd + "/target"}},
  }
  fmt.Println("Building Image " + contextImageName)
  err2 := client.BuildImage(opts)
  check(err2)
  fmt.Println("Build Output: ", buf.String())
}

func removeImage(client *docker.Client, imageName string) {
  err := client.RemoveImage(imageName)
  check(err)
}

func saveAndLoadContextImage(client *docker.Client, contextImageName string) {
  reader, writer := io.Pipe()
  errChan := make(chan error)
  go func() {
    loadOptions := docker.LoadImageOptions {
      InputStream: reader,
    }
    errChan <- client.LoadImage(loadOptions)
  }()

  // Import the image
  exportOptions := docker.ExportImageOptions {
    Name: contextImageName,
    OutputStream: writer,
  }
  err := client.ExportImage(exportOptions)
  check(err)
  writer.Close()
  err = <-errChan
  check(err)
}

func build(client *docker.Client, conf ClusterConfig) {
  fmt.Println("Building Context image...")
  pullImage(client, conf.BaseImageName)
  writeDockerfile(conf.BaseImageName)
  buildTestContextImage(client, conf.ContextImageName)
  // Distribute image within swarm
  saveAndLoadContextImage(client, conf.ContextImageName)
}

func waitForContainer(client *docker.Client, containerId string) {
  fmt.Println("Waiting for container " + containerId)
  code, err := client.WaitContainer(containerId)
  check(err)
  fmt.Println("Container " + containerId + " terminated with code " + string(code))
}

func execute(client *docker.Client, conf ClusterConfig) {
  var data = createTar()
  var containerId = createContainer(client, conf, data)
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
      // Build & distribute image
      buildAndDistImage(conf)
      createImageDoneFile()
      return true
    }
  }

  func execTest(conf ClusterConfig) {
    client, _ := docker.NewClient(conf.Dockerswarm)
    execute(client, conf)
  }

  func buildAndDistImage(conf ClusterConfig) {
    client, _ := docker.NewClient(conf.Dockerswarm)
    build(client, conf);
  }

  func clean(conf ClusterConfig) {
    client, _ := docker.NewClient(conf.Dockerswarm)
    err := client.RemoveImage(conf.ContextImageName)
    check(err)
  }

  func main() {
    var conf = readConfig()

    var command string
    var jarFlag string
    flag.StringVar(&jarFlag, "jar", "", "Surefire arguments")
    flag.StringVar(&command, "c", "help", "Command to be executed [help|run|build|tar|cleanup]")
    flag.Parse()

    switch command {
    case "build":
      buildAndDistImage(conf)
      break
    case "clean":
      clean(conf)
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
      default:
        flag.PrintDefaults()
      }
    }
