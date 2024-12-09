package installer

import (
	"fmt"
	"os/exec"

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

	cmd := exec.Command(
		"debootstrap",
		"--arch="+i.Config.Installation.Architecture,
		i.Config.Installation.DebianVersion,
		i.Config.Installation.MountPoint,
		"http://deb.debian.org/debian",
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install base system: %v", err)
	}

	return nil
}

func (i *Installer) configureSystem() error {
	i.Logger.Info("Configuring system")

	if err := i.generateFstab(); err != nil {
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

	if err := i.installAdditionalPackages(); err != nil {
		return err
	}

	if err := i.installBootloader(); err != nil {
		return err
	}

	return nil
}
