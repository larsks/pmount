package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

type Partition struct {
	Device string
	Number int
	Size   string
}

type SfdiskPartition struct {
	Node  string `json:"node"`
	Start int    `json:"start"`
	Size  int    `json:"size"`
	Type  string `json:"type"`
}

type SfdiskPartitionTable struct {
	Label      string            `json:"label"`
	ID         string            `json:"id"`
	Device     string            `json:"device"`
	Unit       string            `json:"unit"`
	SectorSize int               `json:"sectorsize"`
	Partitions []SfdiskPartition `json:"partitions"`
}

type SfdiskOutput struct {
	PartitionTable SfdiskPartitionTable `json:"partitiontable"`
}

type MountManager struct {
	sourceDevice      string
	targetDir         string
	nbdDevice         string
	partitions        []Partition
	logger            *log.Logger
	format            string
	nbdDeviceExplicit string
}

func NewMountManager(sourceDevice, targetDir, format, nbdDevice string) *MountManager {
	return &MountManager{
		sourceDevice:      sourceDevice,
		targetDir:         targetDir,
		format:            format,
		nbdDeviceExplicit: nbdDevice,
		logger:            log.New(os.Stdout, "[pmount] ", log.LstdFlags),
	}
}

func (mm *MountManager) isImageFile() bool {
	info, err := os.Stat(mm.sourceDevice)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

func (mm *MountManager) discoverMountedPartitions() error {
	// First, find partition directories in the target directory
	entries, err := os.ReadDir(mm.targetDir)
	if err != nil {
		return fmt.Errorf("failed to read target directory: %w", err)
	}

	mm.partitions = []Partition{}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "partition") {
			continue
		}

		partitionPath := filepath.Join(mm.targetDir, entry.Name())

		// Use findmnt to check if this partition directory is mounted
		cmd := exec.Command("findmnt", "-J", "-M", partitionPath)
		output, err := cmd.Output()
		if err != nil {
			// Directory not mounted, skip
			continue
		}

		// Parse findmnt JSON output to get mounted device
		var findmntData struct {
			Filesystems []struct {
				Target string `json:"target"`
				Source string `json:"source"`
			} `json:"filesystems"`
		}

		if err := json.Unmarshal(output, &findmntData); err != nil {
			mm.logger.Printf("warning: failed to parse findmnt output for %s: %v", partitionPath, err)
			continue
		}

		if len(findmntData.Filesystems) > 0 {
			// Extract partition number from directory name (partition1 -> 1)
			partName := entry.Name()
			if len(partName) > 9 { // "partition" is 9 chars
				if partNum := partName[9:]; partNum != "" {
					// Convert to int for consistency
					var num int
					if _, err := fmt.Sscanf(partNum, "%d", &num); err != nil {
						return fmt.Errorf("failed to read partition number from %s", partNum)
					}

					partition := Partition{
						Device: findmntData.Filesystems[0].Source,
						Number: num,
						Size:   "unknown", // We don't need size for unmounting
					}
					mm.partitions = append(mm.partitions, partition)
				}
			}
		}
	}

	mm.logger.Printf("discovered %d mounted partitions", len(mm.partitions))
	return nil
}

func (mm *MountManager) extractNBDDevice() {
	// Look through partitions to find NBD device
	for _, partition := range mm.partitions {
		if strings.Contains(partition.Device, "/dev/nbd") {
			// Extract NBD device from partition device (e.g., /dev/nbd1p1 -> /dev/nbd1)
			if idx := strings.LastIndex(partition.Device, "p"); idx > 0 {
				mm.nbdDevice = partition.Device[:idx]
				mm.logger.Printf("detected NBD device: %s", mm.nbdDevice)
				return
			}
		}
	}
}

func (mm *MountManager) findFreeNBDDevice() (string, error) {
	for i := range 16 {
		nbdDevice := fmt.Sprintf("/dev/nbd%d", i)

		// Check if NBD device exists
		if _, err := os.Stat(nbdDevice); os.IsNotExist(err) {
			continue
		}

		// Check if device is in use by looking for the pid file
		pidFile := fmt.Sprintf("/sys/class/block/nbd%d/pid", i)
		if _, err := os.Stat(pidFile); err == nil {
			// If pid file exists, NBD device is in use
			continue
		}

		// Device appears free
		return nbdDevice, nil
	}

	return "", fmt.Errorf("no free NBD devices found (checked /dev/nbd0 through /dev/nbd15)")
}

func (mm *MountManager) attachImageWithNBD() error {
	var nbdDevice string
	var err error

	if mm.nbdDeviceExplicit != "" {
		nbdDevice = mm.nbdDeviceExplicit
		mm.logger.Printf("using explicitly specified NBD device: %s", nbdDevice)
	} else {
		nbdDevice, err = mm.findFreeNBDDevice()
		if err != nil {
			return fmt.Errorf("failed to find free NBD device: %w", err)
		}
		mm.logger.Printf("using discovered NBD device: %s", nbdDevice)
	}

	args := []string{"--connect=" + nbdDevice}
	if mm.format != "" {
		args = append(args, "--format="+mm.format)
	}
	args = append(args, mm.sourceDevice)

	cmd := exec.Command("qemu-nbd", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to attach image with qemu-nbd: %w\nOutput: %s", err, string(output))
	}
	mm.nbdDevice = nbdDevice
	mm.logger.Printf("attached %s to %s", mm.sourceDevice, mm.nbdDevice)
	return nil
}

func (mm *MountManager) detachNBD() error {
	if mm.nbdDevice == "" {
		return nil
	}
	cmd := exec.Command("qemu-nbd", "--disconnect", mm.nbdDevice)
	if output, err := cmd.CombinedOutput(); err != nil {
		mm.logger.Printf("warning: failed to disconnect NBD device %s: %v\nOutput: %s", mm.nbdDevice, err, string(output))
		return err
	}
	mm.logger.Printf("disconnected NBD device %s", mm.nbdDevice)
	mm.nbdDevice = ""
	return nil
}

func (mm *MountManager) getActiveDevice() string {
	if mm.nbdDevice != "" {
		return mm.nbdDevice
	}
	return mm.sourceDevice
}

func (mm *MountManager) discoverPartitions() error {
	device := mm.getActiveDevice()
	cmd := exec.Command("sfdisk", "-J", device)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list partitions: %w", err)
	}

	var sfdiskData SfdiskOutput
	if err := json.Unmarshal(output, &sfdiskData); err != nil {
		return fmt.Errorf("failed to parse sfdisk output: %w", err)
	}

	mm.partitions = []Partition{}
	for i, part := range sfdiskData.PartitionTable.Partitions {
		// Calculate size in human readable format (sectors to bytes to MB/GB)
		sizeBytes := part.Size * sfdiskData.PartitionTable.SectorSize
		var sizeStr string
		if sizeBytes >= 1024*1024*1024 {
			sizeStr = fmt.Sprintf("%.1fG", float64(sizeBytes)/(1024*1024*1024))
		} else if sizeBytes >= 1024*1024 {
			sizeStr = fmt.Sprintf("%.1fM", float64(sizeBytes)/(1024*1024))
		} else {
			sizeStr = fmt.Sprintf("%dK", sizeBytes/1024)
		}

		partition := Partition{
			Device: part.Node,
			Number: i + 1,
			Size:   sizeStr,
		}
		mm.partitions = append(mm.partitions, partition)
	}

	mm.logger.Printf("discovered %d partitions", len(mm.partitions))
	return nil
}

func (mm *MountManager) createTargetDirectories() error {
	if err := os.MkdirAll(mm.targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", mm.targetDir, err)
	}

	for _, partition := range mm.partitions {
		partDir := filepath.Join(mm.targetDir, fmt.Sprintf("partition%d", partition.Number))
		if err := os.MkdirAll(partDir, 0755); err != nil {
			return fmt.Errorf("failed to create partition directory %s: %w", partDir, err)
		}
	}
	return nil
}

func (mm *MountManager) mountPartitions() error {
	for _, partition := range mm.partitions {
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

func (mm *MountManager) unmountPartitions() error {
	for _, partition := range mm.partitions {
		partDir := filepath.Join(mm.targetDir, fmt.Sprintf("partition%d", partition.Number))

		cmd := exec.Command("umount", partDir)
		if err := cmd.Run(); err != nil {
			mm.logger.Printf("failed to unmount %s: %v", partDir, err)
			continue
		}
		mm.logger.Printf("unmounted %s", partDir)
	}
	return nil
}

func (mm *MountManager) removePartitionDirectories() error {
	for _, partition := range mm.partitions {
		partDir := filepath.Join(mm.targetDir, fmt.Sprintf("partition%d", partition.Number))
		if err := os.Remove(partDir); err != nil {
			mm.logger.Printf("failed to remove directory %s: %v", partDir, err)
		}
	}
	return nil
}

func (mm *MountManager) Mount() error {
	if mm.isImageFile() {
		if err := mm.attachImageWithNBD(); err != nil {
			return err
		}
	}

	if err := mm.discoverPartitions(); err != nil {
		return err
	}

	if len(mm.partitions) == 0 {
		mm.logger.Printf("no partitions found on %s", mm.getActiveDevice())
		return nil
	}

	if err := mm.createTargetDirectories(); err != nil {
		return err
	}

	return mm.mountPartitions()
}

func (mm *MountManager) Unmount() error {
	if err := mm.discoverMountedPartitions(); err != nil {
		return fmt.Errorf("failed to discover mounted partitions: %w", err)
	}

	// Extract NBD device if any partitions are on NBD
	mm.extractNBDDevice()

	if err := mm.unmountPartitions(); err != nil {
		return err
	}

	if err := mm.removePartitionDirectories(); err != nil {
		return err
	}

	if mm.nbdDevice != "" {
		return mm.detachNBD()
	}

	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] <device_or_image> <target_directory>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "       %s --unmount <target_directory>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s /dev/sdb /mnt/usb\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s disk.img /mnt/image\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --format qcow2 disk.qcow2 /mnt/image\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --nbd-device /dev/nbd2 disk.img /mnt/image\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --unmount /mnt/usb\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	pflag.PrintDefaults()
}

func main() {
	// Define command line flags
	var unmount = pflag.BoolP("unmount", "u", false, "unmount partitions and clean up")
	var format = pflag.StringP("format", "f", "", "image format for qemu-nbd (e.g., qcow2, raw, vmdk)")
	var nbdDevice = pflag.StringP("nbd-device", "d", "", "specify NBD device to use (e.g., /dev/nbd1)")
	var help = pflag.BoolP("help", "h", false, "show this help message")

	pflag.Parse()

	// Handle help flag
	if *help {
		printUsage()
		os.Exit(0)
	}

	// Get positional arguments
	args := pflag.Args()

	var device, targetDir string
	if *unmount {
		// For unmount, only target directory is required
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: --unmount requires exactly one argument (target directory)\n")
			printUsage()
			os.Exit(1)
		}
		targetDir = args[0]
		device = "" // Will be discovered from mount state
	} else {
		// For mount, both device and target directory are required
		if len(args) != 2 {
			fmt.Fprintf(os.Stderr, "Error: mount requires exactly two arguments (device and target directory)\n")
			printUsage()
			os.Exit(1)
		}
		device = args[0]
		targetDir = args[1]
	}

	currentUser, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get current user: %v\n", err)
		os.Exit(1)
	}
	if currentUser.Uid != "0" {
		fmt.Fprintf(os.Stderr, "Error: This program must be run as root\n")
		os.Exit(1)
	}

	manager := NewMountManager(device, targetDir, *format, *nbdDevice)
	if *unmount {
		err = manager.Unmount()
	} else {
		err = manager.Mount()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
