# pmount - Partition Mount Utility

A program that automatically mounts all partitions from a block device or disk image to separate directories. That is, given a disk image that looks like this:

```bash
$ fdisk -l disk.img
Disk disk.img: 2.73 GiB, 2927624192 bytes, 5718016 sectors
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes
Disklabel type: dos
Disk identifier: 0x7351b90c

Device     Boot   Start     End Sectors  Size Id Type
disk.img1         16384 1064959 1048576  512M  c W95 FAT32 (LBA)
disk.img2       1064960 5718015 4653056  2.2G 83 Linux
```

Running:

```
sudo pmount --format raw disk.img /mnt
```

Results in:

```
$ mount | grep /mnt
/dev/nbd1p1 on /mnt/partition1 type vfat (rw,relatime,fmask=0022,dmask=0022,codepage=437,iocharset=ascii,shortname=mixed,errors=remount-ro)
/dev/nbd1p2 on /mnt/partition2 type ext4 (rw,relatime,seclabel)
```

Pmount supports any disk format supported by `qemu-nbd` (which includes raw disk images, qcow2, vmdk, and others).

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

## Building

```bash
make
```

## Testing

```bash
make test
```
