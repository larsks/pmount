package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/spf13/pflag"

	mm "github.com/larsks/pmount/internal/mountmanager"
)

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

	manager := mm.NewMountManager(device, targetDir, *format, *nbdDevice)
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
