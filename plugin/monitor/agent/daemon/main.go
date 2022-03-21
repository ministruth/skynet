package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"skynet/sn/utils"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ztrue/tracerr"
)

func main() {
	self, err := filepath.Abs(os.Args[0])
	if err != nil {
		log.Fatalf("Daemon: Failed to get path, error: %v", err)
	}
	var args []string
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}

	dir := filepath.Dir(self)
	target := dir
	if runtime.GOOS == "windows" {
		log.Fatal("Platform not supported")
	} else {
		target = filepath.Join(target, "agent")
	}
	os.Remove(target + ".new") // remove broken file

	for {
		cmd := exec.Command(target, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err = tracerr.Wrap(cmd.Run()); err != nil {
			log.Fatalf("Daemon: Failed to launch, error: %v", err)
		}
		if utils.FileExist(target + ".new") {
			log.Info("Daemon: Updating...")
			os.Rename(target+".new", target)
		} else {
			log.Warn("Daemon: Agent exits, restarting...")
			time.Sleep(1 * time.Second)
		}
	}
}
