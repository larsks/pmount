# pmount - Partition Mount Utility

A program that automatically mounts all partitions from a block device or disk image to separate directories.

## Features

- Mount all partitions from a block device to individual directories
- Supports a variety of disk image formats using `qemu-nbd`

## Requirements

- Go 1.24.5 or later
- Root privileges (required for mounting)
- `qemu-nbd` (for disk image support)
- `sfdisk` (for partition discovery)

## Usage

### Mounting a block device:
```bash
sudo ./pmount /dev/sdb /mnt/usb
```

### Mounting a disk image:
```bash
sudo ./pmount disk.img /mnt/image
sudo ./pmount --format qcow2 disk.qcow2 /mnt/image
sudo ./pmount --format vmdk disk.vmdk /mnt/image
```

### Unmounting:
```bash
sudo ./pmount --unmount /dev/sdb /mnt/usb
sudo ./pmount --unmount disk.img /mnt/image
```

## How it works

1. **Disk images**: Attaches the image using `qemu-nbd`, then discovers partitions
2. **Block devices**: Uses `sfdisk` to discover partitions on the device
3. **Directory structure**: Creates `partition1`, `partition2`, etc. subdirectories in the target directory
4. **Mounting**: Attempts to mount each partition, logging failures without stopping
5. **Cleanup**: On unmount, discovers currently mounted partitions, extracts NBD device information from mount sources, unmounts all partitions, removes per-partition directories (but preserves the target directory), and disconnects NBD connections

## Building

```bash
go build -o pmount .
```

## Testing

```bash
go test -v
```

## Security Note

This program requires root privileges to mount filesystems. Always verify the source device and target directory before running.
