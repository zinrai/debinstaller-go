package config

import (
	"gopkg.in/yaml.v2"
	"os"
)

type PartitionType string

const (
	PartitionTypeBiosBoot  PartitionType = "bios_boot"
	PartitionTypeEfiSystem PartitionType = "efi_system"
	PartitionTypeBoot      PartitionType = "boot"
	PartitionTypeLvmPV     PartitionType = "lvm_pv"
)

type LogicalVolume struct {
	Name       string `yaml:"name"`
	Size       string `yaml:"size"`
	Filesystem string `yaml:"filesystem"`
	MountPoint string `yaml:"mount_point"`
}

type Partition struct {
	Type           PartitionType   `yaml:"type"`
	Size           string          `yaml:"size"`
	Filesystem     string          `yaml:"filesystem,omitempty"`
	MountPoint     string          `yaml:"mount_point,omitempty"`
	VolumeGroup    string          `yaml:"volume_group,omitempty"`
	LogicalVolumes []LogicalVolume `yaml:"logical_volumes,omitempty"`
}

type Config struct {
	Storage struct {
		Devices    []string `yaml:"devices"`
		Bootloader struct {
			Type string `yaml:"type"`
		} `yaml:"bootloader"`
		Partitions []Partition `yaml:"partitions"`
	} `yaml:"storage"`
	System struct {
		Hostname string `yaml:"hostname"`
		Locale   string `yaml:"locale,omitempty"`
	} `yaml:"system"`
	Network struct {
		Interface string `yaml:"interface"`
		IPAddress string `yaml:"ip_address"`
	} `yaml:"network"`
	Users []struct {
		Username string   `yaml:"username"`
		Password string   `yaml:"password"`
		Groups   []string `yaml:"groups"`
	} `yaml:"users"`
	Packages     []string `yaml:"packages"`
	Installation struct {
		MountPoint    string `yaml:"mount_point"`
		Architecture  string `yaml:"architecture"`
		DebianVersion string `yaml:"debian_version"`
	} `yaml:"installation"`
	LogFile string `yaml:"log_file"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
