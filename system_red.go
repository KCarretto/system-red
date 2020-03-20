package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
)

const systemRedRoot = "/system_red"

func init() {
	log.Println("[System Red] Initializing...")
	reexec.Register("/sbin/init", initNamespace)
	if reexec.Init() {
		log.Println("[System Red] Initialized")
		os.Exit(0)
	}
}

func initNamespace() {
	if err := tryProcRemount(); err != nil {
		fmt.Printf("Error mounting /proc - %s\n", err)
		os.Exit(1)
	}
	// if err := mountProc(systemRedRoot); err != nil {
	// 	fmt.Printf("Error mounting /proc - %s\n", err)
	// 	os.Exit(1)
	// }

	// if err := pivotRoot(systemRedRoot); err != nil {
	// 	fmt.Printf("Error running pivot_root - %s\n", err)
	// 	os.Exit(1)
	// }

	run()
}

func run() {
	// initPath, err := filepath.EvalSymlinks("/sbin/init")
	// if err != nil {
	// 	log.Printf("Error resolving init filepath\n", err)
	// }
	// cmd := exec.Command(initPath, "--system")
	cmd := exec.Command("/bin/sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{"PS1=-[ns-process]- # "}

	if err := cmd.Run(); err != nil {
		log.Printf("Error running system init - %s\n", err)
		os.Exit(1)
	}
}

func main() {
	// exitIfRootfsNotFound(systemRedRoot)

	cmd := reexec.Command("/sbin/init")

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID,
		// Noctty:     true,
	}
	if err := cmd.Run(); err != nil {
		log.Printf("Error running the command - %s\n", err)
		os.Exit(1)
	}
}
