package main

import (
	"log"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/docker/docker/pkg/reexec"
)

const systemRedRoot = "/system_red"

func init() {
	log.Println("[System Red] Registering Namespace")
	reexec.Register("/sbin/init", initNamespace)
	if reexec.Init() {
		os.Exit(0)
	}
}

func initNamespace() {
	// if err := tryProcRemount(); err != nil {
	// 	fmt.Printf("Error mounting /proc - %s\n", err)
	// 	os.Exit(1)
	// }
	// if err := mountSystemdCGroup(); err != nil {
	// 	fmt.Printf("Error mounting systemd cgroup - %s\n", err)
	// 	os.Exit(1)
	// }

	log.Println("[System Red] Preparing Container")
	if err := option2(); err != nil {
		log.Printf("[System Red] Error initializing namespace: %s", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	// if err := pivotRoot(systemRedRoot); err != nil {
	// 	fmt.Printf("Error running pivot_root - %s\n", err)
	// 	os.Exit(1)
	// }

	if err := run(); err != nil {
		log.Printf("[System Red] Exited with error: %s\n", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
}

func run() error {
	// initPath, err := filepath.EvalSymlinks("/sbin/init")
	// if err != nil {
	// 	log.Printf("Error resolving init filepath\n", err)
	// }
	// cmd := exec.Command(initPath, "--system")
	// cmd.Env = []string{"container=lxc"}

	// cmd := exec.Command("/lib/systemd/systemd", "--system")
	// cmd.Env = []string{"container=lxc"}
	// sh, err := filepath.EvalSymlinks("/bin/sh")
	// if err != nil {
	// 	return fmt.Errorf("failed to resolve path for /bin/sh: %w", err)
	// }
	// cmd := exec.Command("exec", "/bin/sh")

	// cmd.Stdin = os.Stdin
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	args := []string{"/sbin/init"}
	env := os.Environ()
	log.Println("[System Red] Entering Container")
	if err := syscall.Exec("/lib/systemd/systemd", args, env); err != nil {
		panic(err)
	}
	return nil
}

func main() {
	// exitIfRootfsNotFound(systemRedRoot)
	log.Println("[System Red] Initializing...")
	// if err := createRootfs(systemRedRoot); err != nil {
	// 	log.Printf("[System Red] failed to create rootfs: %s", err)
	// 	os.Exit(1)
	// }

	cmd := reexec.Command("/sbin/init")

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: unix.CLONE_NEWNS |
			unix.CLONE_NEWUTS |
			unix.CLONE_NEWIPC |
			unix.CLONE_NEWCGROUP |
			unix.CLONE_NEWPID,
		// Noctty:     true,
	}
	if err := cmd.Run(); err != nil {
		log.Printf("Error running the command - %s\n", err)
		os.Exit(1)
	}
}
