package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	imageFilename = "base.qcow2"
)

type Config struct {
	isoFilepath string
	imageSize   string
	image       string
	kvm         bool
	compress    bool
}

func newConfigFromFlags() (c Config) {
	flag.StringVar(&c.isoFilepath, "iso", "", "path to operating system iso file")
	flag.StringVar(&c.imageSize, "size", "50G", "size of the virtual machine image")
	flag.StringVar(&c.image, "image", "", "name of the Docker image")
	flag.BoolVar(&c.kvm, "kvm", false, "enable KVM full virtualization support")
	flag.BoolVar(&c.compress, "compress", false, "compress virtual machine image after installation")
	flag.Parse()
	if c.isoFilepath == "" || c.image == "" {
		flag.Usage()
		os.Exit(1)
	}
	return
}

type Installer struct {
	config        Config
	contextDir    string
	imageFilepath string
}

func newInstaller(config Config) (i Installer, err error) {
	i.config = config
	i.contextDir, err = ioutil.TempDir("", "docker-context")
	i.imageFilepath = filepath.Join(i.contextDir, imageFilename)
	return
}

func (i *Installer) Install() error {
	log.Printf("Context dir: %s\n", i.contextDir)
	defer os.RemoveAll(i.contextDir)

	log.Println("Creating machine image...")
	output, err := i.createImage()
	if err != nil {
		log.Print(output)
		return err
	}

	log.Println("Starting machine...")
	output, err = i.runMachine()
	if err != nil {
		log.Print(output)
		return err
	}

	if i.config.compress {
		log.Println("Compressing image...")
		if output, err := i.compressImage(); err != nil {
			log.Print(output)
			return err
		}
	}

	log.Println("Building Docker image...")
	output, err = i.buildImage()
	if err != nil {
		log.Print(output)
		return err
	}

	log.Println("Pushing Docker image...")
	output, err = i.pushImage()
	if err != nil {
		log.Print(output)
		return err
	}

	return nil
}

func (i *Installer) createImage() (string, error) {
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", i.imageFilepath, i.config.imageSize)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func (i *Installer) compressImage() (string, error) {
	tempFilepath := i.imageFilepath + ".temp"
	cmd := exec.Command("qemu-img", "convert", "-O", "qcow2", "-c", i.imageFilepath, tempFilepath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return out.String(), err
	}

	err = os.Rename(tempFilepath, i.imageFilepath)
	return out.String(), err
}

func (i *Installer) buildImage() (string, error) {
	if err := i.writeDockerfile(); err != nil {
		return "", err
	}

	cmd := exec.Command("docker", "build", "-t", i.config.image, i.contextDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func (i *Installer) pushImage() (string, error) {
	cmd := exec.Command("docker", "push", i.config.image)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func (i *Installer) writeDockerfile() error {
	dockerFilepath := filepath.Join(i.contextDir, "Dockerfile")
	imageFilename := filepath.Base(i.imageFilepath)
	dockerfileContents := fmt.Sprintf("FROM busybox\nCOPY %s /base_image/", imageFilename)
	return ioutil.WriteFile(dockerFilepath, []byte(dockerfileContents), 0777)
}

func (i *Installer) runMachine() (string, error) {
	cmd := exec.Command("qemu-system-x86_64",
		"-m", "size=1024",
		"-smp", "cpus=1",
		"-cdrom", i.config.isoFilepath,
		"-vnc", "0.0.0.0:0",
		"-drive", fmt.Sprintf("file=%s", i.imageFilepath))
	if i.config.kvm {
		cmd.Args = append([]string{cmd.Args[0], "-enable-kvm"}, cmd.Args[1:]...)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Start()
	if err != nil {
		return out.String(), err
	}

	go func() {
		reader := bufio.NewReader(os.Stdin)
		log.Println("Press [enter] when installation is complete.")
		_, _ = reader.ReadString('\n')

		err = cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Fatal(err)
		}
	}()

	err = cmd.Wait()
	return out.String(), err
}
