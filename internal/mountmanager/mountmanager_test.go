package mountmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewMountManager(t *testing.T) {
	mm, err := NewMountManager("/dev/sda", "/mnt/test", "", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}

	if mm.sourceDevice != "/dev/sda" {
		t.Errorf("Expected sourceDevice to be /dev/sda, got %s", mm.sourceDevice)
	}

	if mm.targetDir != "/mnt/test" {
		t.Errorf("Expected targetDir to be /mnt/test, got %s", mm.targetDir)
	}

	if mm.format != "" {
		t.Errorf("Expected format to be empty, got %s", mm.format)
	}

	if mm.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if mm.profileName != "default" {
		t.Errorf("Expected profileName to be default, got %s", mm.profileName)
	}

	if mm.profile == nil {
		t.Error("Expected profile to be initialized")
	}
}

func TestNewMountManagerWithFormat(t *testing.T) {
	mm, err := NewMountManager("/path/to/disk.qcow2", "/mnt/test", "qcow2", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}

	if mm.sourceDevice != "/path/to/disk.qcow2" {
		t.Errorf("Expected sourceDevice to be /path/to/disk.qcow2, got %s", mm.sourceDevice)
	}

	if mm.targetDir != "/mnt/test" {
		t.Errorf("Expected targetDir to be /mnt/test, got %s", mm.targetDir)
	}

	if mm.format != "qcow2" {
		t.Errorf("Expected format to be qcow2, got %s", mm.format)
	}
}

func TestNBDDeviceSelection(t *testing.T) {
	// Test automatic NBD device discovery
	mm, err := NewMountManager("/path/to/disk.img", "/mnt/test", "raw", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}
	if mm.nbdDeviceExplicit != "" {
		t.Errorf("Expected nbdDeviceExplicit to be empty for auto discovery, got %s", mm.nbdDeviceExplicit)
	}

	// Test explicit NBD device
	mm2, err := NewMountManager("/path/to/disk.img", "/mnt/test", "raw", "/dev/nbd5", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}
	if mm2.nbdDeviceExplicit != "/dev/nbd5" {
		t.Errorf("Expected nbdDeviceExplicit to be /dev/nbd5, got %s", mm2.nbdDeviceExplicit)
	}
}

func TestIsImageFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.img")
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	f.Close()

	mm, err := NewMountManager(testFile, "/mnt/test", "", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}
	if !mm.isImageFile() {
		t.Error("Expected isImageFile to return true for regular file")
	}

	// Test with non-existent file
	mm2, err := NewMountManager("/dev/nonexistent", "/mnt/test", "", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}
	if mm2.isImageFile() {
		t.Error("Expected isImageFile to return false for non-existent file")
	}
}

func TestCreateTargetDirectories(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "mount_target")

	mm, err := NewMountManager("/dev/test", targetDir, "", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}
	mm.partitions = []Partition{
		{Device: "/dev/test1", Number: 1, Size: "1G"},
		{Device: "/dev/test2", Number: 2, Size: "2G"},
	}

	err = mm.createTargetDirectories()
	if err != nil {
		t.Fatalf("Failed to create target directories: %v", err)
	}

	// Check if main target directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Error("Target directory was not created")
	}

	// Check if partition directories exist
	for _, partition := range mm.partitions {
		partDir := filepath.Join(targetDir, fmt.Sprintf("partition%d", partition.Number))
		if _, err := os.Stat(partDir); os.IsNotExist(err) {
			t.Errorf("Partition directory %s was not created", partDir)
		}
	}
}

func TestRemovePartitionDirectories(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "mount_target")

	// Create directories first
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	partDir1 := filepath.Join(targetDir, "partition1")
	partDir2 := filepath.Join(targetDir, "partition2")

	err = os.MkdirAll(partDir1, 0755)
	if err != nil {
		t.Fatalf("Failed to create partition1 directory: %v", err)
	}

	err = os.MkdirAll(partDir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create partition2 directory: %v", err)
	}

	mm, err := NewMountManager("/dev/test", targetDir, "", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}
	mm.partitions = []Partition{
		{Device: "/dev/test1", Number: 1, Size: "1G"},
		{Device: "/dev/test2", Number: 2, Size: "2G"},
	}

	err = mm.removePartitionDirectories()
	if err != nil {
		t.Fatalf("Failed to remove partition directories: %v", err)
	}

	// Check if partition directories were removed
	if _, err := os.Stat(partDir1); !os.IsNotExist(err) {
		t.Error("Partition1 directory was not removed")
	}

	if _, err := os.Stat(partDir2); !os.IsNotExist(err) {
		t.Error("Partition2 directory was not removed")
	}

	// Check that target directory still exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Error("Target directory should not have been removed")
	}
}

func TestGetActiveDevice(t *testing.T) {
	mm, err := NewMountManager("/dev/sda", "/mnt/test", "", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}

	// Test without NBD device
	if mm.getActiveDevice() != "/dev/sda" {
		t.Errorf("Expected active device to be /dev/sda, got %s", mm.getActiveDevice())
	}

	// Test with NBD device
	mm.nbdDevice = "/dev/nbd0"
	if mm.getActiveDevice() != "/dev/nbd0" {
		t.Errorf("Expected active device to be /dev/nbd0, got %s", mm.getActiveDevice())
	}
}

func TestExtractNBDDevice(t *testing.T) {
	mm, err := NewMountManager("/dev/test", "/mnt/test", "", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}

	// Test with NBD partitions
	mm.partitions = []Partition{
		{Device: "/dev/nbd1p1", Number: 1, Size: "1G"},
		{Device: "/dev/nbd1p2", Number: 2, Size: "2G"},
	}

	mm.extractNBDDevice()
	if mm.nbdDevice != "/dev/nbd1" {
		t.Errorf("Expected NBD device to be /dev/nbd1, got %s", mm.nbdDevice)
	}

	// Test with non-NBD partitions
	mm2, err := NewMountManager("/dev/test", "/mnt/test", "", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}
	mm2.partitions = []Partition{
		{Device: "/dev/sda1", Number: 1, Size: "1G"},
		{Device: "/dev/sda2", Number: 2, Size: "2G"},
	}

	mm2.extractNBDDevice()
	if mm2.nbdDevice != "" {
		t.Errorf("Expected no NBD device for non-NBD partitions, got %s", mm2.nbdDevice)
	}
}

func TestDiscoverMountedPartitions(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "mount_target")

	// Create target directory and partition subdirectories
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	partDir1 := filepath.Join(targetDir, "partition1")
	partDir2 := filepath.Join(targetDir, "partition2")
	nonPartDir := filepath.Join(targetDir, "other")

	for _, dir := range []string{partDir1, partDir2, nonPartDir} {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	mm, err := NewMountManager("/dev/test", targetDir, "", "", "default")
	if err != nil {
		t.Fatalf("NewMountManager() error = %v", err)
	}

	// This will call findmnt on each partition directory
	// Since these aren't actually mounted, we expect 0 partitions discovered
	err = mm.discoverMountedPartitions()
	if err != nil {
		t.Fatalf("discoverMountedPartitions failed: %v", err)
	}

	// Should find 0 mounted partitions since nothing is actually mounted
	if len(mm.partitions) != 0 {
		t.Errorf("Expected 0 mounted partitions (since not actually mounted), got %d", len(mm.partitions))
	}
}

func TestPartitionStruct(t *testing.T) {
	p := Partition{
		Device: "/dev/sda1",
		Number: 1,
		Size:   "500M",
	}

	if p.Device != "/dev/sda1" {
		t.Errorf("Expected device to be /dev/sda1, got %s", p.Device)
	}

	if p.Number != 1 {
		t.Errorf("Expected partition number to be 1, got %d", p.Number)
	}

	if p.Size != "500M" {
		t.Errorf("Expected size to be 500M, got %s", p.Size)
	}
}

func TestSfdiskStructs(t *testing.T) {
	// Test that our sfdisk structs are properly structured
	sfdisk := SfdiskOutput{
		PartitionTable: SfdiskPartitionTable{
			Label:      "dos",
			ID:         "0x12345678",
			Device:     "/dev/sda",
			Unit:       "sectors",
			SectorSize: 512,
			Partitions: []SfdiskPartition{
				{
					Node:  "/dev/sda1",
					Start: 2048,
					Size:  1048576,
					Type:  "83",
				},
			},
		},
	}

	if sfdisk.PartitionTable.Device != "/dev/sda" {
		t.Errorf("Expected device to be /dev/sda, got %s", sfdisk.PartitionTable.Device)
	}

	if len(sfdisk.PartitionTable.Partitions) != 1 {
		t.Errorf("Expected 1 partition, got %d", len(sfdisk.PartitionTable.Partitions))
	}

	if sfdisk.PartitionTable.Partitions[0].Node != "/dev/sda1" {
		t.Errorf("Expected partition node to be /dev/sda1, got %s", sfdisk.PartitionTable.Partitions[0].Node)
	}
}
