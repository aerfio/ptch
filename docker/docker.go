package docker

import (
	"log"
	"os/exec"
)

func EnsureRunning() bool {
	return exec.Command("docker", "info").Run() == nil
}
func EnsureInstalled() bool {
	_, err := exec.LookPath("docker")
	if err != nil {
		log.Print(err)
		return false
	}
	return true
}

func SaveImage(image, path string) error {
	return exec.Command("docker", "save", image, "-o", path).Run()
}
