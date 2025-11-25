package mountmanager

import (
	"testing"
)

func TestNewProfile(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		wantErr     bool
		wantType    string
	}{
		{
			name:        "default profile",
			profileName: "default",
			wantErr:     false,
			wantType:    "default",
		},
		{
			name:        "single profile",
			profileName: "single",
			wantErr:     false,
			wantType:    "single",
		},
		{
			name:        "raspberrypi profile",
			profileName: "raspberrypi",
			wantErr:     false,
			wantType:    "raspberrypi",
		},
		{
			name:        "unknown profile",
			profileName: "unknown",
			wantErr:     true,
			wantType:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := NewProfile(tt.profileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && profile.Name() != tt.wantType {
				t.Errorf("NewProfile() profile name = %v, want %v", profile.Name(), tt.wantType)
			}
		})
	}
}

func TestDefaultProfile_Validate(t *testing.T) {
	profile := &DefaultProfile{}

	tests := []struct {
		name       string
		partitions []Partition
		wantErr    bool
	}{
		{
			name:       "no partitions",
			partitions: []Partition{},
			wantErr:    false,
		},
		{
			name: "one partition",
			partitions: []Partition{
				{Device: "/dev/sda1", Number: 1, Size: "1G"},
			},
			wantErr: false,
		},
		{
			name: "multiple partitions",
			partitions: []Partition{
				{Device: "/dev/sda1", Number: 1, Size: "1G"},
				{Device: "/dev/sda2", Number: 2, Size: "2G"},
				{Device: "/dev/sda3", Number: 3, Size: "3G"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := profile.Validate(tt.partitions)
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultProfile.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSingleProfile_Validate(t *testing.T) {
	profile := &SingleProfile{}

	tests := []struct {
		name       string
		partitions []Partition
		wantErr    bool
	}{
		{
			name:       "no partitions",
			partitions: []Partition{},
			wantErr:    true,
		},
		{
			name: "one partition",
			partitions: []Partition{
				{Device: "/dev/sda1", Number: 1, Size: "1G"},
			},
			wantErr: false,
		},
		{
			name: "two partitions",
			partitions: []Partition{
				{Device: "/dev/sda1", Number: 1, Size: "1G"},
				{Device: "/dev/sda2", Number: 2, Size: "2G"},
			},
			wantErr: true,
		},
		{
			name: "three partitions",
			partitions: []Partition{
				{Device: "/dev/sda1", Number: 1, Size: "1G"},
				{Device: "/dev/sda2", Number: 2, Size: "2G"},
				{Device: "/dev/sda3", Number: 3, Size: "3G"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := profile.Validate(tt.partitions)
			if (err != nil) != tt.wantErr {
				t.Errorf("SingleProfile.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRaspberryPiProfile_Validate(t *testing.T) {
	profile := &RaspberryPiProfile{}

	tests := []struct {
		name       string
		partitions []Partition
		wantErr    bool
	}{
		{
			name:       "no partitions",
			partitions: []Partition{},
			wantErr:    true,
		},
		{
			name: "one partition",
			partitions: []Partition{
				{Device: "/dev/sda1", Number: 1, Size: "1G"},
			},
			wantErr: true,
		},
		{
			name: "two partitions",
			partitions: []Partition{
				{Device: "/dev/sda1", Number: 1, Size: "1G"},
				{Device: "/dev/sda2", Number: 2, Size: "2G"},
			},
			wantErr: false,
		},
		{
			name: "three partitions",
			partitions: []Partition{
				{Device: "/dev/sda1", Number: 1, Size: "1G"},
				{Device: "/dev/sda2", Number: 2, Size: "2G"},
				{Device: "/dev/sda3", Number: 3, Size: "3G"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := profile.Validate(tt.partitions)
			if (err != nil) != tt.wantErr {
				t.Errorf("RaspberryPiProfile.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProfile_Names(t *testing.T) {
	tests := []struct {
		profile  MountProfile
		wantName string
	}{
		{
			profile:  &DefaultProfile{},
			wantName: "default",
		},
		{
			profile:  &SingleProfile{},
			wantName: "single",
		},
		{
			profile:  &RaspberryPiProfile{},
			wantName: "raspberrypi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.wantName, func(t *testing.T) {
			if got := tt.profile.Name(); got != tt.wantName {
				t.Errorf("Profile.Name() = %v, want %v", got, tt.wantName)
			}
		})
	}
}
