package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	self, err := filepath.Abs(os.Args[0])
	if err != nil {
		log.Fatalf("Daemon: Failed to get path, error: %v", err)
	}

	dir := filepath.Dir(self)
	target := dir
	target = filepath.Join(target, strings.ReplaceAll(filepath.Base(self), "daemon", "skynet"))

	for {
		cmd := exec.Command(target, "run")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err = cmd.Run(); err != nil {
			log.Printf("Daemon: Program return error: %v", err)
		}
		log.Println("Daemon: Program exits, restarting...")
		time.Sleep(1 * time.Second)
	}
}
