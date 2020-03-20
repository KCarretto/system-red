package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/containers/storage/pkg/mount"
)

func exitIfRootfsNotFound(rootfsPath string) {
	if _, err := os.Stat(rootfsPath); os.IsNotExist(err) {
		log.Printf(`
"%s" does not exist.
Please create this directory and unpack a suitable root filesystem inside it.
An example rootfs, BusyBox, can be downloaded from:
https://raw.githubusercontent.com/teddyking/ns-process/4.0/assets/busybox.tar
And unpacked by:
mkdir -p %s
tar -C %s -xf busybox.tar
`, rootfsPath, rootfsPath, rootfsPath)
		os.Exit(1)
	}
}

func tryProcRemount() error {
	if err := mount.MakeRPrivate("/"); err != nil {
		return fmt.Errorf("failed to make / private in new mount namespace: %w", err)
	}

	unix.Unmount("/proc", unix.MNT_DETACH)

	return mountProc("/")
}

func mountProc(newroot string) error {
	os.MkdirAll("proc", 0755)

	source := "proc"
	target := filepath.Join(newroot, "/proc")
	fstype := "proc"
	flags := 0
	data := ""

	os.MkdirAll(target, 0755)
	if err := syscall.Mount(source, target, fstype, uintptr(flags), data); err != nil {
		return err
	}

	return nil
}

// pivotRoot will call pivot_root such that rootfs becomes the new root
// filesystem, and everything else is cleaned up.
func pivotRoot(rootfs string) error {
	// make everything in new ns private
	if err := mount.MakeRPrivate("/"); err != nil {
		return fmt.Errorf("failed to make / private in new mount namespace: %w", err)
	}

	if mounted, _ := mount.Mounted(rootfs); !mounted {
		if err := mount.Mount(rootfs, rootfs, "bind", "rbind,rw"); err != nil {
			return fmt.Errorf("failed to bind mount rootfs: %w", err)
		}
	}

	// setup oldRoot for pivot_root
	pivotDir, err := ioutil.TempDir(rootfs, ".pivot_root")
	if err != nil {
		return fmt.Errorf("Error setting up pivot dir: %w", err)
	}

	// Cleanup when done
	var mounted bool
	defer func() {
		if mounted {
			// make sure pivotDir is not mounted before we try to remove it
			if errCleanup := unix.Unmount(pivotDir, unix.MNT_DETACH); errCleanup != nil {
				if err == nil {
					err = errCleanup
				}
				return
			}
		}

		errCleanup := os.Remove(pivotDir)
		// pivotDir doesn't exist if pivot_root failed and chroot+chdir was successful
		// because we already cleaned it up on failed pivot_root
		if errCleanup != nil && !os.IsNotExist(errCleanup) {
			errCleanup = fmt.Errorf("Error cleaning up after pivot: %w", errCleanup)
			if err == nil {
				err = errCleanup
			}
		}
	}()

	if err := unix.PivotRoot(rootfs, pivotDir); err != nil {
		return fmt.Errorf("Failed to pivot root: %w", err)
	}
	mounted = true

	// This is the new path for where the old root (prior to the pivot) has been moved to
	// This dir contains the rootfs of the caller, which we need to remove so it is not visible during extraction
	pivotDir = filepath.Join("/", filepath.Base(pivotDir))

	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("Error changing to new root: %v", err)
	}

	// Make the pivotDir (where the old root lives) private so it can be unmounted without propagating to the host
	if err := unix.Mount("", pivotDir, "", unix.MS_PRIVATE|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("Error making old root private after pivot: %v", err)
	}

	// Now unmount the old root so it's no longer visible from the new root
	if err := unix.Unmount(pivotDir, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("Error while unmounting old root after pivot: %v", err)
	}
	mounted = false

	return nil
}
