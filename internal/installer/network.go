package installer

import (
	"fmt"
	"os"
)

func (i *Installer) configureNetwork() error {
	i.Logger.Info("Configuring network")

	interfacesDir := i.Config.Installation.MountPoint + "/etc/network/interfaces.d"

	var networkConfig string
	if i.Config.Network.Type == "dhcp" {
		networkConfig = fmt.Sprintf(`auto %s
iface %s inet dhcp
`, i.Config.Network.Interface, i.Config.Network.Interface)
	} else if i.Config.Network.Type == "static" {
		networkConfig = fmt.Sprintf(`auto %s
iface %s inet static
        address %s
        netmask %s
        gateway %s
`, i.Config.Network.Interface, i.Config.Network.Interface,
			i.Config.Network.Address, i.Config.Network.Netmask, i.Config.Network.Gateway)
	} else {
		return fmt.Errorf("unsupported network type: %s", i.Config.Network.Type)
	}

	if err := os.WriteFile(fmt.Sprintf("%s/%s", interfacesDir, i.Config.Network.Interface),
		[]byte(networkConfig), 0644); err != nil {
		return fmt.Errorf("failed to write interface configuration: %v", err)
	}

	return nil
}
