package mountmanager

import "fmt"

// MountProfile defines the interface for different mount strategies
type MountProfile interface {
	// Validate checks if the discovered partitions meet the profile's requirements
	Validate(partitions []Partition) error

	// Mount executes the profile-specific mount logic
	Mount(mm *MountManager, partitions []Partition) error

	// Unmount executes the profile-specific unmount logic
	Unmount(mm *MountManager) error

	// Name returns the profile name
	Name() string
}

// NewProfile creates a new mount profile by name
func NewProfile(name string) (MountProfile, error) {
	switch name {
	case "default":
		return &DefaultProfile{}, nil
	case "single":
		return &SingleProfile{}, nil
	case "raspberrypi":
		return &RaspberryPiProfile{}, nil
	default:
		return nil, fmt.Errorf("unknown mount profile: %s (valid options: default, single, raspberrypi)", name)
	}
}
