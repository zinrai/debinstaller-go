package installer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zinrai/debinstaller-go/internal/config"
	"github.com/zinrai/debinstaller-go/internal/utils"
)

type Installer struct {
	Config *config.Config
	Logger *utils.Logger
}

func NewInstaller(cfg *config.Config, logger *utils.Logger) *Installer {
	return &Installer{
		Config: cfg,
		Logger: logger,
	}
}

func (i *Installer) Install() error {
	i.Logger.Info("Starting Debian installation")

	if err := i.prepareStorage(); err != nil {
		return fmt.Errorf("failed to prepare storage: %v", err)
	}

	if err := i.installBaseSystem(); err != nil {
		return fmt.Errorf("failed to install base system: %v", err)
	}

	if err := i.configureSystem(); err != nil {
		return fmt.Errorf("failed to configure system: %v", err)
	}

	i.Logger.Info("Debian installation completed successfully")
	return nil
}

func (i *Installer) installBaseSystem() error {
	i.Logger.Info("Installing base system")

	grubPackage := "grub2"
	if i.Config.Storage.Bootloader.Type == "efi" {
		grubPackage = "grub-efi"
	}

	packages := []string{
		"openssh-server",
		"lvm2",
		"sudo",
		"locales",
		grubPackage,
		"linux-image-" + i.Config.Installation.Architecture,
	}

	if err := utils.RunCommand(i.Logger, "debootstrap",
		"--arch="+i.Config.Installation.Architecture,
		"--include="+strings.Join(packages, ","),
		i.Config.Installation.DebianVersion,
		i.Config.Installation.MountPoint,
		"http://deb.debian.org/debian"); err != nil {
		return fmt.Errorf("failed to install base system: %v", err)
	}

	return nil
}

func (i *Installer) mountSpecialFilesystems() error {
	i.Logger.Info("Mounting special filesystems for chroot")

	mountPoints := []struct {
		options []string
		source  string
		target  string
	}{
		{[]string{"--bind"}, "/dev", "/dev"},
		{[]string{"-t", "proc"}, "none", "/proc"},
		{[]string{"--bind"}, "/sys", "/sys"},
	}

	for _, mp := range mountPoints {
		target := filepath.Join(i.Config.Installation.MountPoint, mp.target)
		args := append(mp.options, mp.source, target)
		if err := utils.RunCommand(i.Logger, "mount", args...); err != nil {
			return fmt.Errorf("failed to mount %s: %v", mp.target, err)
		}
	}

	return nil
}

func (i *Installer) configureSystem() error {
	i.Logger.Info("Configuring system")

	if err := i.generateFstab(); err != nil {
		return err
	}

	if err := i.mountSpecialFilesystems(); err != nil {
		return fmt.Errorf("failed to mount special filesystems: %v", err)
	}

	if err := i.installAdditionalPackages(); err != nil {
		return err
	}

	if err := i.setHostname(); err != nil {
		return err
	}

	if err := i.configureLocale(); err != nil {
		return err
	}

	if err := i.configureNetwork(); err != nil {
		return err
	}

	if err := i.configureUsers(); err != nil {
		return err
	}

	if err := i.installBootloader(); err != nil {
		return err
	}

	return nil
}
