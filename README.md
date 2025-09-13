# debinstaller-go

A Go-based Debian installer that supports both BIOS and EFI boot installations with LVM configuration.

## Features

- Support for both BIOS and EFI boot systems
- LVM support
- YAML-based configuration
- Flexible partition configuration
- Automated installation process
- Support for both DHCP and static IP configuration

## Prerequisites

- [Debian Live environment](https://live-team.pages.debian.net/live-manual/)
- Required Debian packages:
  - `gdisk`: partition management with sgdisk command
  - `lvm2`: LVM operations
  - `debootstrap`: base system installation
  - `dosfstools`: vfat filesystem operations with mkfs.vfat command
  - `arch-install-scripts`: generating fstab with genfstab command

## Installation

```bash
$ go build -o debinstaller-go ./cmd/debinstaller/main.go
```

## Usage

1. Create a configuration file (see `./example` for BIOS or EFI boot setup)
2. Run the installer with root privileges:

```bash
$ sudo ./debinstaller-go -config config.yaml
```

## Configuration

The installer uses YAML configuration files. Two example configurations are provided:

- `config.yaml.bios`: BIOS boot systems
- `config.yaml.efi`: EFI boot systems

### Storage Configuration

The storage section defines the partition layout and LVM configuration:

```yaml
storage:
  devices:
    - /dev/sda
  bootloader:
    type: "bios"  # or "efi"
  partitions:
    - type: "bios_boot"      # For BIOS systems only
      size: "2M"
    - type: "efi_system"     # For EFI systems only
      size: "512M"
      filesystem: "vfat"
      mount_point: "/boot/efi"
    - type: "boot"
      size: "512M"
      filesystem: "ext2"
      mount_point: "/boot"
    - type: "lvm_pv"
      size: "15G"
      volume_group: "vg0"
      logical_volumes:
        - name: "root"
          size: "3G"
          filesystem: "ext4"
          mount_point: "/"
        - name: "home"
          size: "1G"
          filesystem: "ext4"
          mount_point: "/home"
        - name: "var"
          size: "1G"
          filesystem: "ext4"
          mount_point: "/var"
```

### Network Configuration

Support for both DHCP and static IP configuration.

DHCP Configuration:

```yaml
network:
  interface: "ens1"
  type: "dhcp"
```

Static IP Configuration:

```yaml
network:
  interface: "ens1"
  type: "static"
  address: "192.168.2.20"
  netmask: "255.255.255.0"
  gateway: "192.168.2.254"
```

### System Configuration

Basic system settings:

```yaml
system:
  hostname: "debian-server"
  locale: "ja_JP.UTF-8"  # Optional. Defaults to C.UTF-8 if not specified

users:
  - username: "admin"
    password: "changeme"
    groups:
      - "sudo"

packages:
  - vim
```

### Installation Settings

Installation-specific configurations:

```yaml
installation:
  mount_point: "/mnt/debian"
  architecture: "amd64"
  debian_version: "bookworm"

log_file: "/tmp/debian_install.log"
```

## License

This project is licensed under the [MIT License](./LICENSE).
