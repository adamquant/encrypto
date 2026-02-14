//go:build darwin

package drive

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"howett.net/plist"
)

// DiskUtilList represents the plist output from `diskutil list -plist`
type DiskUtilList struct {
	AllDisks              []string   `plist:"AllDisks"`
	AllDisksAndPartitions []DiskInfo `plist:"AllDisksAndPartitions"`
}

// DiskInfo represents a disk entry in the plist
type DiskInfo struct {
	Content            string          `plist:"Content"`
	DeviceIdentifier   string          `plist:"DeviceIdentifier"`
	OSInternal         bool            `plist:"OSInternal"`
	Partitions         []PartitionInfo `plist:"Partitions"`
	APFSPhysicalStores []APFSStore     `plist:"APFSPhysicalStores"`
	APFSVolumes        []APFSVolume    `plist:"APFSVolumes"`
	Size               uint64          `plist:"Size"`
}

// PartitionInfo represents a partition entry
type PartitionInfo struct {
	Content          string `plist:"Content"`
	DeviceIdentifier string `plist:"DeviceIdentifier"`
	Size             uint64 `plist:"Size"`
}

// APFSStore represents APFS physical store reference
type APFSStore struct {
	DeviceIdentifier string `plist:"DeviceIdentifier"`
}

// APFSVolume represents an APFS volume
type APFSVolume struct {
	DeviceIdentifier string `plist:"DeviceIdentifier"`
	Size             uint64 `plist:"Size"`
	VolumeName       string `plist:"VolumeName"`
	MountPoint       string `plist:"MountPoint"`
}

// DiskUtilInfo represents detailed info from `diskutil info -plist`
type DiskUtilInfo struct {
	DeviceIdentifier      string `plist:"DeviceIdentifier"`
	DeviceNode            string `plist:"DeviceNode"`
	VolumeName            string `plist:"VolumeName"`
	Size                  uint64 `plist:"TotalSize"`
	Removable             bool   `plist:"RemovableMedia"`
	Ejectable             bool   `plist:"Ejectable"`
	MediaName             string `plist:"MediaName"`
	VolumeKind            string `plist:"VolumeKind"`
	FilesystemPersonality string `plist:"FilesystemPersonality"`
	MountPoint            string `plist:"MountPoint"`
}

func (m *Manager) listDarwin() ([]Drive, error) {
	// Get list of all disks
	cmd := exec.Command("diskutil", "list", "-plist")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run diskutil list: %w", err)
	}

	// Parse plist
	var diskList DiskUtilList
	decoder := plist.NewDecoder(bytes.NewReader(output))
	if err := decoder.Decode(&diskList); err != nil {
		return nil, fmt.Errorf("failed to parse diskutil plist: %w", err)
	}

	var drives []Drive

	// Process each physical disk
	for _, diskInfo := range diskList.AllDisksAndPartitions {
		// Skip if this is a volume/container, not a physical disk
		if strings.HasPrefix(diskInfo.Content, "Apple_APFS") ||
			strings.HasPrefix(diskInfo.Content, "APFS") {
			continue
		}

		// Get detailed info for this disk
		detailedInfo, err := m.getDiskInfo(diskInfo.DeviceIdentifier)
		if err != nil {
			// Log error but continue with other disks
			continue
		}

		drive := Drive{
			Name:        detailedInfo.MediaName,
			Path:        fmt.Sprintf("/dev/%s", diskInfo.DeviceIdentifier),
			Size:        detailedInfo.Size,
			Type:        determineDriveType(detailedInfo),
			IsRemovable: detailedInfo.Removable || detailedInfo.Ejectable,
		}

		// If media name is empty, use device identifier
		if drive.Name == "" {
			drive.Name = diskInfo.DeviceIdentifier
		}

		drives = append(drives, drive)
	}

	return drives, nil
}

func (m *Manager) getDiskInfo(deviceID string) (*DiskUtilInfo, error) {
	cmd := exec.Command("diskutil", "info", "-plist", deviceID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk info for %s: %w", deviceID, err)
	}

	var info DiskUtilInfo
	decoder := plist.NewDecoder(bytes.NewReader(output))
	if err := decoder.Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse disk info plist: %w", err)
	}

	return &info, nil
}

func determineDriveType(info *DiskUtilInfo) DriveType {
	if info.Removable || info.Ejectable {
		return DriveTypeRemovable
	}
	return DriveTypeFixed
}
