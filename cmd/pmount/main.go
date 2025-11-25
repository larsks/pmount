package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/spf13/pflag"

	mm "github.com/larsks/pmount/internal/mountmanager"
)

type (
	Options struct {
		unmount   bool
		format    string
		nbdDevice string
		profile   string
		help      bool
	}
)

var options Options

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] <device_or_image> <target_directory>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "       %s --unmount <target_directory>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s /dev/sdb /mnt/usb\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s disk.img /mnt/image\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --format qcow2 disk.qcow2 /mnt/image\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --nbd-device /dev/nbd2 disk.img /mnt/image\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --profile single single-partition.img /mnt/image\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --profile raspberrypi raspios.img /mnt/rpi\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --unmount --profile single /mnt/image\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --unmount /mnt/usb\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	pflag.PrintDefaults()
}

func init() {
	// Define command line flags
	pflag.BoolVarP(&options.unmount, "unmount", "u", false, "unmount partitions and clean up")
	pflag.BoolVarP(&options.unmount, "umount", "", false, "unmount partitions and clean up")
	pflag.StringVarP(&options.format, "format", "f", "", "image format for qemu-nbd (e.g., qcow2, raw, vmdk)")
	pflag.StringVarP(&options.nbdDevice, "nbd-device", "d", "", "specify NBD device to use (e.g., /dev/nbd1)")
	pflag.StringVarP(&options.profile, "profile", "p", "default", "mount profile to use (default, single, raspberrypi)")
	pflag.BoolVarP(&options.help, "help", "h", false, "show this help message")
}

func main() {
	pflag.Parse()

	// Handle help flag
	if options.help {
		printUsage()
		os.Exit(0)
	}

	// Get positional arguments
	args := pflag.Args()

	var device, targetDir string
	if options.unmount {
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

	currentUser, userErr := user.Current()
	if userErr != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get current user: %v\n", userErr)
		os.Exit(1)
	}
	if currentUser.Uid != "0" {
		fmt.Fprintf(os.Stderr, "Error: This program must be run as root\n")
		os.Exit(1)
	}

	manager, err := mm.NewMountManager(device, targetDir, options.format, options.nbdDevice, options.profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if options.unmount {
		err = manager.Unmount()
	} else {
		err = manager.Mount()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
