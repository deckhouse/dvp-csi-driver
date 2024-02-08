package mounter

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	mu "k8s.io/mount-utils"
	utilexec "k8s.io/utils/exec"
)

/* DEPS:
mount - already in alpine
umount - already in alpine
blkid - from blkid
findmnt - from findmnt
fsck - already in alpine
mkfs.ext4 - from e2fsprogs
mkfs.xfs - from xfsprogs
*/

type Mounter struct {
	logger *slog.Logger
	mutils mu.SafeFormatAndMount
}

// New returns a new mounter instance.
func New(logger *slog.Logger) *Mounter {
	return &Mounter{
		logger: logger,
		mutils: mu.SafeFormatAndMount{
			Interface: mu.New("/bin/mount"),
			Exec:      utilexec.New(),
		},
	}
}

func (m *Mounter) MountFileSystem(source, target, fsType string, opts ...string) error {
	switch fsType {
	case "ext4", "xfs":
	case "":
		fsType = "ext4"
		m.logger.Debug("Got empty fs type: set the default value", "fs-type", fsType)
	default:
		return fmt.Errorf("got unsupported fs type: %s", fsType)
	}

	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to stat source device: %w", err)
	}

	if (info.Mode() & os.ModeDevice) != os.ModeDevice {
		return fmt.Errorf("[NewMount] path %s is not a device", source)
	}

	err = os.MkdirAll(target, os.FileMode(0o755))
	if err != nil {
		return fmt.Errorf("could not create target directory %s: %w", target, err)
	}

	_, err = m.mutils.IsMountPoint(target)
	if err != nil {
		return fmt.Errorf("unable to determine mount status of %s %w", target, err)
	}

	err = m.mutils.FormatAndMount(source, target, fsType, opts)
	if err != nil {
		return fmt.Errorf("failed to FormatAndMount : %w", err)
	}

	return nil
}

func (m *Mounter) MountBlockDevice(source, target string, opts ...string) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to stat source device: %w", err)
	}

	if (info.Mode() & os.ModeDevice) != os.ModeDevice {
		return fmt.Errorf("[NewMount] path %s is not a device", source)
	}

	f, err := os.OpenFile(target, os.O_CREATE, os.FileMode(0o666))
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("could not create bind target for block volume %s, %w", target, err)
		}
	} else {
		_ = f.Close()
	}

	err = m.mutils.Mount(source, target, "", append(opts, "bind"))
	if err != nil {
		return err
	}

	return nil
}

func (m *Mounter) Unmount(target string) error {
	err := m.mutils.Unmount(target)
	if err != nil {
		return err
	}

	return nil
}

const devicesByIDPath = "/dev/disk/by-id/"

func (m *Mounter) GetBlockDevicePathByID(id string) (string, error) {
	symlinks, err := os.ReadDir(devicesByIDPath)
	if err != nil {
		return "", err
	}

	var symlinkName string
	for _, symlink := range symlinks {
		if strings.HasSuffix(symlink.Name(), id) {
			symlinkName = symlink.Name()
			break
		}
	}

	if symlinkName == "" {
		return "", errors.New("symlink not found")
	}

	blockDevicePath, err := filepath.EvalSymlinks(devicesByIDPath + symlinkName)
	if err != nil {
		return "", err
	}

	return blockDevicePath, nil
}
