package installer

import (
	"fmt"
	"os"

	"github.com/zinrai/debinstaller-go/internal/utils"
)

func (i *Installer) generateFstab() error {
	i.Logger.Info("Generating fstab")

	fstabContent, err := utils.RunCommandWithOutput(i.Logger, "genfstab", "-U", i.Config.Installation.MountPoint)
	if err != nil {
		return fmt.Errorf("failed to generate fstab: %v", err)
	}

	if err := os.WriteFile(i.Config.Installation.MountPoint+"/etc/fstab", fstabContent, 0644); err != nil {
		return fmt.Errorf("failed to write fstab: %v", err)
	}

	return nil
}

func (i *Installer) setHostname() error {
	i.Logger.Info("Setting hostname")

	if err := os.WriteFile(i.Config.Installation.MountPoint+"/etc/hostname", []byte(i.Config.System.Hostname), 0644); err != nil {
		return fmt.Errorf("failed to set hostname: %v", err)
	}

	return nil
}

func (i *Installer) configureLocale() error {
	i.Logger.Info("Configuring locale")

	// C.UTF-8 is available by default, so do not run locale-gen.
	locale := "C.UTF-8"
	if i.Config.System.Locale != "" && i.Config.System.Locale != "C.UTF-8" {
		// If the set locale is not C.UTF-8, execute locale-gen.
		if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint, "locale-gen", i.Config.System.Locale); err != nil {
			return fmt.Errorf("failed to generate locale: %v", err)
		}
		locale = i.Config.System.Locale
	}

	if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint, "update-locale", "LANG="+locale); err != nil {
		return fmt.Errorf("failed to update locale: %v", err)
	}

	return nil
}

func (i *Installer) configureUsers() error {
	i.Logger.Info("Configuring users")

	for _, user := range i.Config.Users {
		if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint,
			"useradd", "-m", "-s", "/bin/bash", user.Username); err != nil {
			return fmt.Errorf("failed to create user %s: %v", user.Username, err)
		}

		if err := utils.RunCommandWithInput(i.Logger,
			fmt.Sprintf("%s:%s", user.Username, user.Password),
			"chroot", i.Config.Installation.MountPoint, "chpasswd"); err != nil {
			return fmt.Errorf("failed to set password for user %s: %v", user.Username, err)
		}

		for _, group := range user.Groups {
			if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint,
				"gpasswd", "-a", user.Username, group); err != nil {
				return fmt.Errorf("failed to add user %s to group %s: %v", user.Username, group, err)
			}
		}
	}

	return nil
}

func (i *Installer) installAdditionalPackages() error {
	i.Logger.Info("Installing additional packages")

	if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint, "apt-get", "update"); err != nil {
		return fmt.Errorf("failed to update package lists: %v", err)
	}

	args := append([]string{i.Config.Installation.MountPoint, "apt-get", "install", "-y"}, i.Config.Packages...)
	if err := utils.RunCommand(i.Logger, "chroot", args...); err != nil {
		return fmt.Errorf("failed to install additional packages: %v", err)
	}

	return nil
}

func (i *Installer) installBootloader() error {
	i.Logger.Info("Installing bootloader")

	if i.Config.Bootloader.Type == "grub" {
		if i.Config.Bootloader.EFI {
			if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint,
				"grub-install", "--target=x86_64-efi", "--efi-directory=/boot/efi", "--bootloader-id=debian"); err != nil {
				return fmt.Errorf("failed to install GRUB: %v", err)
			}
		} else {
			if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint,
				"grub-install", "--target=i386-pc", i.Config.Storage.Devices[0]); err != nil {
				return fmt.Errorf("failed to install GRUB: %v", err)
			}
		}

		if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint, "update-grub"); err != nil {
			return fmt.Errorf("failed to update GRUB: %v", err)
		}
	} else {
		return fmt.Errorf("unsupported bootloader type: %s", i.Config.Bootloader.Type)
	}

	return nil
}
