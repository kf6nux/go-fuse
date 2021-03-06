package fuse

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func openFUSEDevice() (*os.File, error) {
	fs, err := filepath.Glob("/dev/osxfuse*")
	if err != nil {
		return nil, err
	}
	if len(fs) == 0 {
		// TODO(hanwen): run the load_osxfuse command.
		return nil, fmt.Errorf("no FUSE devices found")
	}
	for _, fn := range fs {
		f, err := os.OpenFile(fn, os.O_RDWR, 0)
		if err != nil {
			continue
		}
		return f, nil
	}

	return nil, fmt.Errorf("all FUSE devices busy")
}

const bin = "/Library/Filesystems/osxfusefs.fs/Support/mount_osxfusefs"

func mount(mountPoint string, opts *MountOptions, ready chan<- error) (fd int, err error) {
	f, err := openFUSEDevice()
	if err != nil {
		return 0, err
	}

	cmd := exec.Command(bin, "-o", strings.Join(opts.optionsStrings(), ","), "-o", fmt.Sprintf("iosize=%d", opts.MaxWrite), "3", mountPoint)
	cmd.ExtraFiles = []*os.File{f}
	cmd.Env = append(os.Environ(), "MOUNT_FUSEFS_CALL_BY_LIB=", "MOUNT_OSXFUSE_CALL_BY_LIB=",
		"MOUNT_OSXFUSE_DAEMON_PATH="+os.Args[0],
		"MOUNT_FUSEFS_DAEMON_PATH="+os.Args[0])

	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Start(); err != nil {
		f.Close()
		return 0, err
	}
	go func() {
		err := cmd.Wait()
		if err != nil {
			err = fmt.Errorf("mount_osxfusefs failed: %v. Stderr: %s, Stdout: %s", err, errOut.String(), out.String())
		}

		ready <- err
		close(ready)
	}()
	return int(f.Fd()), nil
}

func unmount(dir string) error {
	return syscall.Unmount(dir, 0)
}
