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

func option2() error {
	log.Println("[System Red] Initializing Namespace")
	// Mark everything in new mount ns private
	if err := mount.MakeRPrivate("/"); err != nil {
		return fmt.Errorf("failed to make / private in new mount namespace: %w", err)
	}

	log.Println("[System Red] Mount marked as private")

	// Unmount special filesystems
	if err := mount.RecursiveUnmount("/sys"); err != nil {
		return fmt.Errorf("failed to unmount /sys: %w", err)
	}
	log.Println("[System Red] Unmounted /sys")
	if err := mount.RecursiveUnmount("/dev"); err != nil {
		return fmt.Errorf("failed to unmount /dev: %w", err)
	}
	log.Println("[System Red] Unmounted /dev")
	if err := mount.RecursiveUnmount("/run"); err != nil {
		return fmt.Errorf("failed to unmount /run: %w", err)
	}
	log.Println("[System Red] Unmounted /run")
	if err := mount.RecursiveUnmount("/proc"); err != nil {
		return fmt.Errorf("failed to unmount /proc: %w", err)
	}
	log.Println("[System Red] Unmounted /proc")

	// Remount proc first
	procFlags := 0
	if err := syscall.Mount("proc", "/proc", "proc", uintptr(procFlags), ""); err != nil {
		return fmt.Errorf("failed to mount /proc: %w", err)
	}
	log.Println("[System Red] Remounted /proc")

	// Mount SysFS (With SystemD CGroup)
	if err := mount.Mount("sysfs", "/sys", "sysfs", "ro"); err != nil {
		return fmt.Errorf("failed to mount (as rw) /sys: %w", err)
	}
	log.Println("[System Red] Remounted /sys")
	if err := mount.Mount("tmpfs", "/sys/fs/cgroup", "tmpfs", "rw,nosuid,nodev,noexec,mode=755"); err != nil {
		return fmt.Errorf("failed to mount (as rw) /sys/fs/cgroup: %w", err)
	}
	log.Println("[System Red] Remounted /sys/fs/cgroup")
	if err := os.MkdirAll("/sys/fs/cgroup/systemd", 0555); err != nil {
		return fmt.Errorf("failed to create /sys/fs/cgroup/systemd cgroup directory: %w", err)
	}
	log.Println("[System Red] Created systemd cgroup directory at /sys/fs/cgroup/systemd")

	// systemDFlags := 0
	// if err := syscall.Mount("cgroup", "/sys/fs/cgroup/systemd", "cgroup", uintptr(systemDFlags), "rw,nosuid,nodev,noexec,xattr,release_agent=/lib/systemd/systemd-cgroups-agent,name=systemd"); err != nil {
	// 	return fmt.Errorf("failed to mount /sys/fs/cgroup/systemd: %w", err)
	// }
	// if err := mount.Mount("cgroup", "/sys/fs/cgroup/systemd", "cgroup", "rw,nosuid,nodev,noexec,xattr,release_agent=/lib/systemd/systemd-cgroups-agent,name=systemd"); err != nil {
	// 	return fmt.Errorf("failed to mount /sys/fs/cgroup/systemd: %w", err)
	// }
	// log.Println("[System Red] Mounted systemd cgroup /sys/fs/cgroup/systemd")
	// if err := mount.Mount("tmpfs", "/sys/fs/cgroup", "tmpfs", "remount,ro,nosuid,nodev,noexec,mode=755"); err != nil {
	// 	return fmt.Errorf("failed to remount (as ro) /sys/fs/cgroup: %w", err)
	// }
	// log.Println("[System Red] Remounted /sys/fs/cgroup ro")
	// if err := mount.Mount("sysfs", "/sys", "sysfs", "remount,ro"); err != nil {
	// 	return fmt.Errorf("failed to remount (as ro) /sys: %w", err)
	// }
	// log.Println("[System Red] Remounted /sys ro")

	// Mount remaining special filesystems
	if err := mount.Mount("udev", "/dev", "devtmpfs", ""); err != nil {
		return fmt.Errorf("failed to remount /dev: %w", err)
	}
	log.Println("[System Red] Remounted /dev")
	if err := mount.Mount("tmpfs", "/run", "tmpfs", ""); err != nil {
		return fmt.Errorf("failed to remount /run: %w", err)
	}
	log.Println("[System Red] Remounted /run")

	return nil
}

func copyRootFS(rootfs string) error {

	// Create RootFS directory
	// Copy files, skipping blacklisted ones
	return nil
}

func pivotRootFS(rootfs string) error {
	// Mark everything in new mount ns private
	if err := mount.MakeRPrivate("/"); err != nil {
		return fmt.Errorf("failed to make / private in new mount namespace: %w", err)
	}

	// Ensure new root is mounted
	if mounted, _ := mount.Mounted(rootfs); !mounted {
		if err := mount.Mount(rootfs, rootfs, "bind", "rbind,rw"); err != nil {
			return fmt.Errorf("failed to bind mount rootfs: %w", err)
		}
	}

	// Setup temporary pivot directory to hold old root
	pivotDir, err := ioutil.TempDir(rootfs, ".pivot_root")
	if err != nil {
		return fmt.Errorf("Error setting up pivot dir: %w", err)
	}

	var pivoted bool
	defer func() {
		// Ensure old root is unmounted if pivot was successful
		if pivoted {
			if err := unix.Unmount(pivotDir, unix.MNT_DETACH); err != nil {
				return
			}
		}

		// Remove temporary pivot directory
		os.Remove(pivotDir)
	}()

	// Change root to rootfs, moving old root to pivot directory
	if err := unix.PivotRoot(rootfs, pivotDir); err != nil {
		return fmt.Errorf("Failed to pivot root: %w", err)
	}
	pivoted = true
	pivotDir = filepath.Join("/", filepath.Base(pivotDir))

	// Change working directory to new root
	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("Error changing to new root: %v", err)
	}

	// Mark old root as private (so it can be unmounted without affecting host)
	if err := unix.Mount("", pivotDir, "", unix.MS_PRIVATE|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to mount old root as private after pivot: %w", err)
	}

	return nil
}

func initRootFS() error {
	// called after pivotRootFS
	// Mount /proc
	// Mount /tmp
	// Mount /run
	// Handle cgroups
	return nil
}

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

func createEphemeralMounts() error {
	mount.RecursiveUnmount("/proc")
	mount.RecursiveUnmount("/sys")
	mount.RecursiveUnmount("/run")
	mount.RecursiveUnmount("/dev")
	mount.RecursiveUnmount("/tmp")

	os.MkdirAll("/proc", 0755)
	os.MkdirAll("/sys", 0755)
	os.MkdirAll("/run", 0755)
	os.MkdirAll("/dev", 0755)
	os.MkdirAll("/tmp", 0755)

	if err := syscall.Mount("sys", "/sys", "sysfs", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to mount /sys: %w", err)
	}
	cgroupFlags := 0
	if err := syscall.Mount("tmpfs", "/sys/fs/cgroup", "tmpfs", uintptr(cgroupFlags), ""); err != nil {
		return fmt.Errorf("failed to mount /sys/fs/cgroup: %w", err)
	}

	systemdFlags := 0
	os.MkdirAll("/sys/fs/cgroup/systemd", 0755)
	if err := syscall.Mount("tmpfs", "/sys/fs/cgroup/systemd", "tmpfs", uintptr(systemdFlags), ""); err != nil {
		return fmt.Errorf("failed to mount /sys/fs/cgroup/systemd: %w", err)
	}

	runFlags := 0
	if err := syscall.Mount("run", "/run", "tmpfs", uintptr(runFlags), ""); err != nil {
		return fmt.Errorf("failed to mount /run: %w", err)
	}

	devFlags := 0
	if err := syscall.Mount("dev", "/dev", "devtmpfs", uintptr(devFlags), ""); err != nil {
		return fmt.Errorf("failed to mount /dev: %w", err)
	}

	tmpFlags := 0
	if err := syscall.Mount("tmp", "/tmp", "tmpfs", uintptr(tmpFlags), ""); err != nil {
		return fmt.Errorf("failed to mount /tmp: %w", err)
	}

	procFlags := 0
	if err := syscall.Mount("proc", "/proc", "proc", uintptr(procFlags), ""); err != nil {
		return fmt.Errorf("failed to mount /proc: %w", err)
	}
	// mount.RecursiveUnmount("/sys/fs/cgroup")
	// unix.Unmount("/proc", unix.MNT_DETACH)
	// unix.Unmount("/sys/fs/cgroup", unix.MNT_DETACH)
	// if err := mountCGroup(); err != nil {
	// 	return fmt.Errorf("failed to mount cgroups: %w", err)
	// }
	return nil
}

func mountProc(newroot string) error {
	// mount.RecursiveUnmount("/proc")

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

func createRootfs(rootfs string) error {
	os.MkdirAll(filepath.Join(rootfs, "/etc"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/bin"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/sbin"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/usr"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/home"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/root"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/opt"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/var"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/run"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/lib"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/lib64"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/sys"), 0755)
	os.MkdirAll(filepath.Join(rootfs, "/tmp"), 0755)

	if err := syscall.Mount("/etc", filepath.Join(rootfs, "/etc"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /etc: %w", err)
	}
	if err := syscall.Mount("/bin", filepath.Join(rootfs, "/bin"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /bin: %w", err)
	}
	if err := syscall.Mount("/sbin", filepath.Join(rootfs, "/sbin"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /sbin: %w", err)
	}
	if err := syscall.Mount("/usr", filepath.Join(rootfs, "/usr"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /usr: %w", err)
	}
	if err := syscall.Mount("/home", filepath.Join(rootfs, "/home"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /home: %w", err)
	}
	if err := syscall.Mount("/root", filepath.Join(rootfs, "/root"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /root: %w", err)
	}
	if err := syscall.Mount("/opt", filepath.Join(rootfs, "/opt"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /opt: %w", err)
	}
	if err := syscall.Mount("/var", filepath.Join(rootfs, "/var"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /var: %w", err)
	}
	if err := syscall.Mount("/run", filepath.Join(rootfs, "/run"), "tmpfs", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /run: %w", err)
	}
	if err := syscall.Mount("/tmp", filepath.Join(rootfs, "/tmp"), "tmpfs", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /tmp: %w", err)
	}
	if err := syscall.Mount("/lib", filepath.Join(rootfs, "/lib"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /lib: %w", err)
	}
	if err := syscall.Mount("/lib64", filepath.Join(rootfs, "/lib64"), "none", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /lib64: %w", err)
	}
	if err := syscall.Mount("/sys", filepath.Join(rootfs, "/sys"), "sysfs", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to mount /sys: %w", err)
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

	// if err := mountProc(rootfs); err != nil {
	// 	return fmt.Errorf("Error mounting /proc: %w", err)
	// }
	// if err := mountRootFS(rootfs); err != nil {
	// 	return fmt.Errorf("failed to mount root fs: %w", err)
	// }

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

	if err := createEphemeralMounts(); err != nil {
		return fmt.Errorf("failed to create ephemeral mounts: %w", err)
	}
	return nil
}
