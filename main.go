package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func main() {
	installed := ensureDockerInstalled()
	if !installed {
		log.Fatal("Install docker cli before proceeding")
	}
	running := ensureDockerRunning()
	if !running {
		log.Fatal("Start docker before proceeding")
	}
	images, err := listLocalImages()
	if err != nil {
		log.Fatal(err)
	}
	for _, image := range images {
		fmt.Println(image)
	}
}

func ensureDockerInstalled() bool {
	_, err := exec.LookPath("docker")
	if err != nil {
		log.Print(err)
		return false
	}
	return true
}

func ensureDockerRunning() bool {
	return exec.Command("docker", "info").Run() == nil
}

func listLocalImages() ([]string, error) {
	out, err := exec.Command("docker", "images", "-f", "dangling=false", "--format", "{{ .Repository }}:{{ .Tag }}").CombinedOutput()
	if err != nil {
		return []string{}, err
	}

	return strings.Split(string(out), "\n"), nil
}
 