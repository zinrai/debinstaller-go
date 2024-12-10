package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/zinrai/debinstaller-go/internal/config"
	"github.com/zinrai/debinstaller-go/internal/utils"
)

func (i *Installer) prepareStorage() error {
	i.Logger.Info("Preparing storage")

	for _, device := range i.Config.Storage.Devices {
		if err := i.partitionDevice(device); err != nil {
			return err
		}
	}

	if err := i.setupLVM(); err != nil {
		return err
	}

	if err := i.createFilesystems(); err != nil {
		return err
	}

	if err := i.mountFilesystems(); err != nil {
		return err
	}

	return nil
}

func (i *Installer) partitionDevice(device string) error {
	i.Logger.Info("Partitioning device: %s", device)

	// Clear partition table
	if err := utils.RunCommand(i.Logger, "sgdisk", "-Z", "-o", device); err != nil {
		return fmt.Errorf("failed to clear partition table: %v", err)
	}

	args := []string{device}
	for idx, partition := range i.Config.Storage.Partitions {
		partNum := idx + 1

		// Add arguments for partition creation
		args = append(args, "-n", fmt.Sprintf("%d::+%s", partNum, partition.Size))

		// Set the partition type
		typeCode := getPartitionTypeCode(partition.Type)
		args = append(args, "-t", fmt.Sprintf("%d:%s", partNum, typeCode))
	}

	// Execute partitioning
	if err := utils.RunCommand(i.Logger, "sgdisk", args...); err != nil {
		return fmt.Errorf("failed to create partitions: %v", err)
	}

	time.Sleep(time.Second)
	return nil
}

func getPartitionTypeCode(pType config.PartitionType) string {
	switch pType {
	case config.PartitionTypeBiosBoot:
		return "ef02"
	case config.PartitionTypeEfiSystem:
		return "ef00"
	case config.PartitionTypeLvmPV:
		return "8e00"
	default:
		return "8300"
	}
}

func (i *Installer) setupLVM() error {
	i.Logger.Info("Setting up LVM")

	// Find LVM PV partition
	var lvmPartition *config.Partition
	var partitionNumber int
	for idx, part := range i.Config.Storage.Partitions {
		if part.Type == config.PartitionTypeLvmPV {
			lvmPartition = &i.Config.Storage.Partitions[idx]
			partitionNumber = idx + 1
			break
		}
	}

	if lvmPartition == nil {
		return nil // No LVM setup needed
	}

	// Get PV device path
	pvDevice := fmt.Sprintf("%s%d", i.Config.Storage.Devices[0], partitionNumber)

	// Remove existing VG if any
	if err := utils.RunCommand(i.Logger, "vgremove", "-f", lvmPartition.VolumeGroup); err != nil {
		i.Logger.Info("No existing volume group to remove")
	}

	// Remove existing PV if any
	if err := utils.RunCommand(i.Logger, "pvremove", "-ff", pvDevice); err != nil {
		i.Logger.Info("No existing physical volume to remove")
	}

	// Create PV
	if err := utils.RunCommand(i.Logger, "pvcreate", "-ff", pvDevice); err != nil {
		return fmt.Errorf("failed to create physical volume: %v", err)
	}

	// Create VG
	if err := utils.RunCommand(i.Logger, "vgcreate", lvmPartition.VolumeGroup, pvDevice); err != nil {
		return fmt.Errorf("failed to create volume group: %v", err)
	}

	// Create LVs
	for _, lv := range lvmPartition.LogicalVolumes {
		if err := utils.RunCommand(i.Logger, "lvcreate", "-y", "-L", lv.Size,
			"-n", lv.Name, lvmPartition.VolumeGroup); err != nil {
			return fmt.Errorf("failed to create logical volume: %v", err)
		}
	}

	return nil
}

func (i *Installer) createFilesystems() error {
	i.Logger.Info("Creating filesystems")

	// Create filesystems for regular partitions
	for idx, partition := range i.Config.Storage.Partitions {
		if partition.Filesystem == "" || partition.Type == config.PartitionTypeBiosBoot {
			continue
		}

		if partition.Type != config.PartitionTypeLvmPV {
			device := fmt.Sprintf("%s%d", i.Config.Storage.Devices[0], idx+1)
			if err := createFilesystem(i.Logger, partition.Filesystem, device); err != nil {
				return err
			}
		}
	}

	// Create filesystems for logical volumes
	for _, partition := range i.Config.Storage.Partitions {
		if partition.Type != config.PartitionTypeLvmPV {
			continue
		}

		for _, lv := range partition.LogicalVolumes {
			device := fmt.Sprintf("/dev/%s/%s", partition.VolumeGroup, lv.Name)
			if err := createFilesystem(i.Logger, lv.Filesystem, device); err != nil {
				return err
			}
		}
	}

	return nil
}

func createFilesystem(logger *utils.Logger, fsType, device string) error {
	var args []string
	switch fsType {
	case "vfat":
		args = []string{"-F32", device}
	default:
		args = []string{device}
	}

	if err := utils.RunCommand(logger, "mkfs."+fsType, args...); err != nil {
		return fmt.Errorf("failed to create filesystem: %v", err)
	}
	return nil
}

func (i *Installer) mountFilesystems() error {
	i.Logger.Info("Mounting filesystems")

	// Collect mount points
	var mounts []struct {
		device     string
		mountPoint string
	}

	// Add regular partitions
	for idx, partition := range i.Config.Storage.Partitions {
		if partition.MountPoint != "" && partition.Type != config.PartitionTypeLvmPV {
			mounts = append(mounts, struct {
				device     string
				mountPoint string
			}{
				device:     fmt.Sprintf("%s%d", i.Config.Storage.Devices[0], idx+1),
				mountPoint: partition.MountPoint,
			})
		}
	}

	// Add logical volumes
	for _, partition := range i.Config.Storage.Partitions {
		if partition.Type != config.PartitionTypeLvmPV {
			continue
		}

		for _, lv := range partition.LogicalVolumes {
			mounts = append(mounts, struct {
				device     string
				mountPoint string
			}{
				device:     fmt.Sprintf("/dev/%s/%s", partition.VolumeGroup, lv.Name),
				mountPoint: lv.MountPoint,
			})
		}
	}

	// Sort mounts by mount point length to ensure proper order
	sort.Slice(mounts, func(i, j int) bool {
		return len(mounts[i].mountPoint) < len(mounts[j].mountPoint)
	})

	// Mount filesystems
	for _, mount := range mounts {
		mountPoint := filepath.Join(i.Config.Installation.MountPoint, mount.mountPoint)
		if err := os.MkdirAll(mountPoint, 0755); err != nil {
			return fmt.Errorf("failed to create mount point directory: %v", err)
		}

		if err := utils.RunCommand(i.Logger, "mount", mount.device, mountPoint); err != nil {
			return fmt.Errorf("failed to mount filesystem: %v", err)
		}
	}

	return nil
}
