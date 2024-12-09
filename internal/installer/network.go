package installer

import (
	"fmt"
	"os"
)

func (i *Installer) configureNetwork() error {
	i.Logger.Info("Configuring network")

	networkConfig := fmt.Sprintf(`
auto lo
iface lo inet loopback

auto %s
iface %s inet %s
`, i.Config.Network.Interface, i.Config.Network.Interface, i.Config.Network.IPAddress)

	if err := os.WriteFile(i.Config.Installation.MountPoint+"/etc/network/interfaces", []byte(networkConfig), 0644); err != nil {
		return fmt.Errorf("failed to configure network: %v", err)
	}

	return nil
}
