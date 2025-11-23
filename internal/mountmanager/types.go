package mountmanager

import (
	"log"
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
