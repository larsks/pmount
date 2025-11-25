package mountmanager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DefaultProfile implements the default mount behavior:
// - Creates partition1, partition2, etc. subdirectories
// - Mounts each partition to its corresponding subdirectory
type DefaultProfile struct{}

func (p *DefaultProfile) Name() string {
	return "default"
}

func (p *DefaultProfile) Validate(partitions []Partition) error {
	// Default profile works with any number of partitions
	return nil
}

func (p *DefaultProfile) Mount(mm *MountManager, partitions []Partition) error {
	// Create base target directory
	if err := os.MkdirAll(mm.targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", mm.targetDir, err)
	}

	// Create partition subdirectories
	for _, partition := range partitions {
		partDir := filepath.Join(mm.targetDir, fmt.Sprintf("partition%d", partition.Number))
		if err := os.MkdirAll(partDir, 0755); err != nil {
			return fmt.Errorf("failed to create partition directory %s: %w", partDir, err)
		}
	}

	// Mount each partition
	for _, partition := range partitions {
		partDir := filepath.Join(mm.targetDir, fmt.Sprintf("partition%d", partition.Number))

		cmd := exec.Command("mount", partition.Device, partDir)
		if err := cmd.Run(); err != nil {
			mm.logger.Printf("failed to mount %s to %s: %v", partition.Device, partDir, err)
			continue
		}
		mm.logger.Printf("mounted %s (%s) to %s", partition.Device, partition.Size, partDir)
	}
	return nil
}

func (p *DefaultProfile) Unmount(mm *MountManager) error {
	// Discover mounted partitions using default profile structure
	if err := mm.discoverMountedPartitions(); err != nil {
		return fmt.Errorf("failed to discover mounted partitions: %w", err)
	}

	// Unmount each partition
	for _, partition := range mm.partitions {
		partDir := filepath.Join(mm.targetDir, fmt.Sprintf("partition%d", partition.Number))

		cmd := exec.Command("umount", partDir)
		if err := cmd.Run(); err != nil {
			mm.logger.Printf("failed to unmount %s: %v", partDir, err)
			continue
		}
		mm.logger.Printf("unmounted %s", partDir)
	}

	// Remove partition subdirectories
	for _, partition := range mm.partitions {
		partDir := filepath.Join(mm.targetDir, fmt.Sprintf("partition%d", partition.Number))
		if err := os.Remove(partDir); err != nil {
			mm.logger.Printf("failed to remove directory %s: %v", partDir, err)
		}
	}

	return nil
}

// SingleProfile implements single partition mount behavior:
// - Requires exactly one partition
// - Mounts partition directly on target directory (no subdirectory)
type SingleProfile struct{}

func (p *SingleProfile) Name() string {
	return "single"
}

func (p *SingleProfile) Validate(partitions []Partition) error {
	if len(partitions) != 1 {
		return fmt.Errorf("single profile requires exactly 1 partition, found %d", len(partitions))
	}
	return nil
}

func (p *SingleProfile) Mount(mm *MountManager, partitions []Partition) error {
	// Create target directory
	if err := os.MkdirAll(mm.targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", mm.targetDir, err)
	}

	// Mount the single partition directly on target
	partition := partitions[0]
	cmd := exec.Command("mount", partition.Device, mm.targetDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount %s to %s: %w", partition.Device, mm.targetDir, err)
	}
	mm.logger.Printf("mounted %s (%s) to %s", partition.Device, partition.Size, mm.targetDir)
	return nil
}

func (p *SingleProfile) Unmount(mm *MountManager) error {
	// Discover what's mounted on target directory
	device, err := mm.findMountedDevice(mm.targetDir)
	if err != nil {
		return fmt.Errorf("failed to discover mounted device: %w", err)
	}
	if device != "" {
		// Add to partitions list for NBD detection
		mm.addPartition(device, 1)
	}

	// Unmount from target directory
	cmd := exec.Command("umount", mm.targetDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unmount %s: %w", mm.targetDir, err)
	}
	mm.logger.Printf("unmounted %s", mm.targetDir)
	return nil
}

// RaspberryPiProfile implements Raspberry Pi SD card mount behavior:
// - Requires exactly two partitions
// - Mounts partition 2 (root) on target directory
// - Mounts partition 1 (boot) on target/boot/firmware
type RaspberryPiProfile struct{}

func (p *RaspberryPiProfile) Name() string {
	return "raspberrypi"
}

func (p *RaspberryPiProfile) Validate(partitions []Partition) error {
	if len(partitions) != 2 {
		return fmt.Errorf("raspberrypi profile requires exactly 2 partitions, found %d", len(partitions))
	}
	return nil
}

func (p *RaspberryPiProfile) Mount(mm *MountManager, partitions []Partition) error {
	// Create target directory
	if err := os.MkdirAll(mm.targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", mm.targetDir, err)
	}

	// Find partition 1 and partition 2
	var partition1, partition2 *Partition
	for i := range partitions {
		if partitions[i].Number == 1 { //nolint:staticcheck
			partition1 = &partitions[i]
		} else if partitions[i].Number == 2 {
			partition2 = &partitions[i]
		}
	}

	if partition1 == nil || partition2 == nil {
		return fmt.Errorf("could not find both partition 1 and partition 2")
	}

	// Mount partition 2 (root) on target directory
	cmd := exec.Command("mount", partition2.Device, mm.targetDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount %s to %s: %w", partition2.Device, mm.targetDir, err)
	}
	mm.logger.Printf("mounted %s (%s) to %s", partition2.Device, partition2.Size, mm.targetDir)

	// Check if /boot/firmware exists in the mounted filesystem
	bootFirmwarePath := filepath.Join(mm.targetDir, "boot", "firmware")
	if _, err := os.Stat(bootFirmwarePath); os.IsNotExist(err) {
		// Unmount partition 2 before returning error
		exec.Command("umount", mm.targetDir).Run() //nolint:errcheck
		return fmt.Errorf("/boot/firmware directory does not exist in partition 2 filesystem")
	}

	// Mount partition 1 (boot) on target/boot/firmware
	cmd = exec.Command("mount", partition1.Device, bootFirmwarePath)
	if err := cmd.Run(); err != nil {
		// Unmount partition 2 before returning error
		exec.Command("umount", mm.targetDir).Run() //nolint:errcheck
		return fmt.Errorf("failed to mount %s to %s: %w", partition1.Device, bootFirmwarePath, err)
	}
	mm.logger.Printf("mounted %s (%s) to %s", partition1.Device, partition1.Size, bootFirmwarePath)

	return nil
}

func (p *RaspberryPiProfile) Unmount(mm *MountManager) error {
	// Unmount in reverse order: boot partition first, then root partition
	bootFirmwarePath := filepath.Join(mm.targetDir, "boot", "firmware")

	// Discover what's mounted on boot/firmware
	bootDevice, err := mm.findMountedDevice(bootFirmwarePath)
	if err != nil {
		mm.logger.Printf("warning: failed to discover boot device: %v", err)
	}
	if bootDevice != "" {
		mm.addPartition(bootDevice, 1)
	}

	// Discover what's mounted on target directory
	rootDevice, err := mm.findMountedDevice(mm.targetDir)
	if err != nil {
		mm.logger.Printf("warning: failed to discover root device: %v", err)
	}
	if rootDevice != "" {
		mm.addPartition(rootDevice, 2)
	}

	// Unmount boot partition
	cmd := exec.Command("umount", bootFirmwarePath)
	if err := cmd.Run(); err != nil {
		mm.logger.Printf("failed to unmount %s: %v", bootFirmwarePath, err)
		// Continue to try unmounting root partition
	} else {
		mm.logger.Printf("unmounted %s", bootFirmwarePath)
	}

	// Unmount root partition
	cmd = exec.Command("umount", mm.targetDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unmount %s: %w", mm.targetDir, err)
	}
	mm.logger.Printf("unmounted %s", mm.targetDir)

	return nil
}
