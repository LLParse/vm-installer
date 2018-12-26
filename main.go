package main

import (
	"bytes"
	"log"
	"os/exec"
)

var processList = []string{"docker", "qemu-system-x86_64", "qemu-img"}

func getFilepath(processName string) (string, error) {
	cmd := exec.Command("which", processName)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func main() {
	for _, process := range processList {
		_, err := getFilepath(process)
		if err != nil {
			log.Fatal("Missing dependency: ", process)
		}
	}

	config := newConfigFromFlags()
	installer, err := newInstaller(config)
	if err != nil {
		log.Fatal(err)
	}
	if err := installer.Install(); err != nil {
		log.Fatal(err)
	}
	log.Println("Done.")
}
