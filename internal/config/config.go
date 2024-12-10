package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type LVMConfig struct {
	VGName string `yaml:"vg"`
	Size   string `yaml:"size"`
}

type StoragePartition struct {
	MountPoint string `yaml:"mount_point"`
	Size       string `yaml:"size"`
	Filesystem string `yaml:"filesystem"`
	Partition  int    `yaml:"partition"`
	Type       string `yaml:"type,omitempty"`
	LVMConfig  struct {
		VGName string `yaml:"vg,omitempty"`
		Name   string `yaml:"name,omitempty"`
	} `yaml:"lvm,omitempty"`
}

type Config struct {
	Storage struct {
		Devices    []string           `yaml:"devices"`
		LVM        LVMConfig          `yaml:"lvm"`
		Partitions []StoragePartition `yaml:"partitions"`
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
	Packages   []string `yaml:"packages"`
	Bootloader struct {
		EFI bool `yaml:"efi"`
	} `yaml:"bootloader"`
	Advanced struct {
		EnableSerialConsole bool   `yaml:"enable_serial_console"`
		CustomKernelParams  string `yaml:"custom_kernel_params"`
	} `yaml:"advanced"`
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
