package installer

import (
	"fmt"
	"os"
	"strings"

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

	if err := i.configureHosts(); err != nil {
		return fmt.Errorf("failed to configure hosts: %v", err)
	}

	return nil
}

func (i *Installer) configureHosts() error {
	i.Logger.Info("Configuring /etc/hosts")

	hostsPath := i.Config.Installation.MountPoint + "/etc/hosts"

	// Read existing hosts file
	content, err := os.ReadFile(hostsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read hosts file: %v", err)
	}

	// Check if hostname entry already exists
	hostname := i.Config.System.Hostname
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == hostname {
			// Hostname entry already exists
			return nil
		}
	}

	// Prepare new hosts entry
	newEntry := fmt.Sprintf("127.0.1.1\t%s\n", hostname)

	// Append the new entry
	if err := os.WriteFile(hostsPath, []byte(string(content)+newEntry), 0644); err != nil {
		return fmt.Errorf("failed to write hosts file: %v", err)
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

	if i.Config.Storage.Bootloader.Type == "efi" {
		// --removable: UEFI firmware that only loads bootx64.efi from /EFI/BOOT
		if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint,
			"grub-install", "--target=x86_64-efi", "--efi-directory=/boot/efi", "--bootloader-id=debian", "--removable"); err != nil {
			return fmt.Errorf("failed to install GRUB EFI: %v", err)
		}
	} else {
		if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint,
			"grub-install", "--target=i386-pc", i.Config.Storage.Devices[0]); err != nil {
			return fmt.Errorf("failed to install GRUB BIOS: %v", err)
		}
	}

	// Generate grub.cfg
	if err := utils.RunCommand(i.Logger, "chroot", i.Config.Installation.MountPoint,
		"grub-mkconfig", "-o", "/boot/grub/grub.cfg"); err != nil {
		return fmt.Errorf("failed to generate grub.cfg: %v", err)
	}

	return nil
}
