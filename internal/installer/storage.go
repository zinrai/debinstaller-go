package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zinrai/debinstaller-go/internal/config"
	"github.com/zinrai/debinstaller-go/internal/utils"
)

func (i *Installer) prepareStorage() error {
	i.Logger.Info("Preparing storage")

	// partitioning
	for _, device := range i.Config.Storage.Devices {
		if err := i.partitionDevice(device); err != nil {
			return err
		}
	}

	// LVM setup ( if needed )
	if needsLVM(i.Config.Storage.Partitions) {
		if err := i.setupLVM(); err != nil {
			return err
		}
	}

	if err := i.createFilesystems(); err != nil {
		return err
	}

	if err := i.mountFilesystems(); err != nil {
		return err
	}

	return nil
}

func needsLVM(partitions []config.StoragePartition) bool {
	for _, p := range partitions {
		if p.Type == "lvm" {
			return true
		}
	}
	return false
}

func (i *Installer) partitionDevice(device string) error {
	i.Logger.Info("Partitioning device: %s", device)

	// Clear partition table
	if err := utils.RunCommand(i.Logger, "sgdisk", "-Z", "-o", device); err != nil {
		return fmt.Errorf("failed to clear partition table: %v", err)
	}

	// Build arguments for partitioning
	args := []string{device}
	partitionNums := make(map[int]bool)
	lvmPartitionSize := ""

	// Set the size of the LVM partition
	if i.Config.Storage.LVM.Size != "" {
		lvmPartitionSize = i.Config.Storage.LVM.Size
	}

	for _, partition := range i.Config.Storage.Partitions {
		if partitionNums[partition.Partition] {
			continue // The same partition number is processed only once
		}
		partitionNums[partition.Partition] = true

		size := partition.Size
		if partition.Type == "lvm" && lvmPartitionSize != "" {
			size = lvmPartitionSize
		}

		args = append(args, "-n", fmt.Sprintf("%d::+%s", partition.Partition, size))

		// Set the partition type
		typeCode := "8300" // Default is Linux filesystem
		if partition.Filesystem == "vfat" {
			typeCode = "ef00"
		} else if partition.Type == "lvm" {
			typeCode = "8e00" // Linux LVM
		}
		args = append(args, "-t", fmt.Sprintf("%d:%s", partition.Partition, typeCode))
	}

	// Execute partitioning
	if err := utils.RunCommand(i.Logger, "sgdisk", args...); err != nil {
		return fmt.Errorf("failed to create partitions: %v", err)
	}

	time.Sleep(time.Second)
	return nil
}

func (i *Installer) setupLVM() error {
	i.Logger.Info("Setting up LVM")

	// Check the total size of the LVM partitions
	totalSize, err := i.calculateTotalLVMSize()
	if err != nil {
		return fmt.Errorf("failed to calculate total LVM size: %v", err)
	}

	// Compare with the maximum size set
	maxSize, err := parseSize(i.Config.Storage.LVM.Size)
	if err != nil {
		return fmt.Errorf("failed to parse LVM max size: %v", err)
	}

	if totalSize > maxSize {
		return fmt.Errorf("total LVM partition size (%d MB) exceeds maximum size (%d MB)", totalSize/1024/1024, maxSize/1024/1024)
	}

	// Delete existing VGs, if any
	vgName := i.Config.Storage.LVM.VGName
	if err := utils.RunCommand(i.Logger, "vgremove", "-f", vgName); err != nil {
		i.Logger.Info("No existing volume group to remove")
	}

	// Collect device paths for LVM partitions
	var lvmDevices []string
	for _, device := range i.Config.Storage.Devices {
		for _, partition := range i.Config.Storage.Partitions {
			if partition.Type == "lvm" {
				lvmDevice := fmt.Sprintf("%s%d", device, partition.Partition)
				lvmDevices = append(lvmDevices, lvmDevice)
				break // Use only the first LVM partition on each device
			}
		}
	}

	// Delete an existing PV, if any
	for _, device := range lvmDevices {
		if err := utils.RunCommand(i.Logger, "pvremove", "-ff", device); err != nil {
			i.Logger.Info("No existing physical volume to remove on %s", device)
		}
	}

	// Create Physical Volume
	for _, device := range lvmDevices {
		if err := utils.RunCommand(i.Logger, "pvcreate", "-ff", device); err != nil {
			return fmt.Errorf("failed to create physical volume: %v", err)
		}
	}

	// Create Volume Group
	if err := utils.RunCommand(i.Logger, "vgcreate", vgName, strings.Join(lvmDevices, " ")); err != nil {
		return fmt.Errorf("failed to create volume group: %v", err)
	}

	// Create Logical Volume
	for _, partition := range i.Config.Storage.Partitions {
		if partition.Type != "lvm" {
			continue
		}

		// Obtain the LV name
		lvName := partition.LVMConfig.Name
		if lvName == "" {
			return fmt.Errorf("LVM name is required for partition %s", partition.MountPoint)
		}

		if err := utils.RunCommand(i.Logger, "lvcreate", "-y", "-L", partition.Size, "-n", lvName, vgName); err != nil {
			return fmt.Errorf("failed to create logical volume: %v", err)
		}
	}

	return nil
}

func (i *Installer) calculateTotalLVMSize() (int64, error) {
	var totalSize int64
	for _, partition := range i.Config.Storage.Partitions {
		if partition.Type != "lvm" {
			continue
		}
		size, err := parseSize(partition.Size)
		if err != nil {
			return 0, fmt.Errorf("failed to parse size for partition %s: %v", partition.MountPoint, err)
		}
		totalSize += size
	}
	return totalSize, nil
}

func parseSize(sizeStr string) (int64, error) {
	re := regexp.MustCompile(`^(\d+)([GMK])$`)
	matches := re.FindStringSubmatch(sizeStr)
	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	size, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size number: %v", err)
	}

	switch matches[2] {
	case "G":
		size *= 1024 * 1024 * 1024
	case "M":
		size *= 1024 * 1024
	case "K":
		size *= 1024
	}

	return size, nil
}

func (i *Installer) createFilesystems() error {
	i.Logger.Info("Creating filesystems")

	for _, partition := range i.Config.Storage.Partitions {
		device := i.getDevicePath(partition)

		if err := utils.RunCommand(i.Logger, "mkfs."+partition.Filesystem,
			getFsOptions(partition.Filesystem, device)...); err != nil {
			return fmt.Errorf("failed to create filesystem: %v", err)
		}
	}

	return nil
}

func getFsOptions(fsType, device string) []string {
	switch fsType {
	case "vfat":
		return []string{"-F32", device}
	case "ext4":
		return []string{device}
	default:
		return []string{device}
	}
}

func (i *Installer) mountFilesystems() error {
	i.Logger.Info("Mounting filesystems")

	// Determine the order of mount points
	sortedPartitions := sortPartitionsByMountpoint(i.Config.Storage.Partitions)

	for _, partition := range sortedPartitions {
		device := i.getDevicePath(partition)
		mountPoint := filepath.Join(i.Config.Installation.MountPoint, partition.MountPoint)

		if err := os.MkdirAll(mountPoint, 0755); err != nil {
			return fmt.Errorf("failed to create mount point directory: %v", err)
		}

		if err := utils.RunCommand(i.Logger, "mount", device, mountPoint); err != nil {
			return fmt.Errorf("failed to mount filesystem: %v", err)
		}
	}

	return nil
}

func (i *Installer) getDevicePath(partition config.StoragePartition) string {
	if partition.Type == "lvm" {
		vgName := i.Config.Storage.LVM.VGName
		if partition.LVMConfig.VGName != "" {
			vgName = partition.LVMConfig.VGName
		}

		lvName := partition.LVMConfig.Name
		if lvName == "" {
			lvName = strings.ReplaceAll(partition.MountPoint[1:], "/", "-")
		}

		return fmt.Sprintf("/dev/%s/%s", vgName, lvName)
	}
	return fmt.Sprintf("%s%d", i.Config.Storage.Devices[0], partition.Partition)
}

func sortPartitionsByMountpoint(partitions []config.StoragePartition) []config.StoragePartition {
	sorted := make([]config.StoragePartition, len(partitions))
	copy(sorted, partitions)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].MountPoint) < len(sorted[j].MountPoint)
	})
	return sorted
}
