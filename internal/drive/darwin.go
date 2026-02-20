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
	ParentWholeDisk       string `plist:"ParentWholeDisk"`
	WholeDisk             bool   `plist:"WholeDisk"`
	Encrypted             bool   `plist:"Encryption"`
	FileVault             bool   `plist:"FileVault"`
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

	// Build map: APFS physical store partition ID → first APFS volume device ID
	// This lets us associate each physical disk with its APFS volume (e.g. disk5s1)
	apfsVolumeMap := make(map[string]string)
	for _, diskInfo := range diskList.AllDisksAndPartitions {
		if !strings.HasPrefix(diskInfo.Content, "Apple_APFS") {
			continue
		}
		for _, store := range diskInfo.APFSPhysicalStores {
			for _, vol := range diskInfo.APFSVolumes {
				if vol.DeviceIdentifier != "" && apfsVolumeMap[store.DeviceIdentifier] == "" {
					apfsVolumeMap[store.DeviceIdentifier] = vol.DeviceIdentifier
				}
			}
		}
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

		// Try to find a mount point from partitions or APFS volumes
		mountPoint := detailedInfo.MountPoint
		if mountPoint == "" {
			for _, part := range diskInfo.Partitions {
				partInfo, err := m.getDiskInfo(part.DeviceIdentifier)
				if err == nil && partInfo.MountPoint != "" {
					mountPoint = partInfo.MountPoint
					break
				}
			}
		}
		if mountPoint == "" {
			for _, vol := range diskInfo.APFSVolumes {
				if vol.MountPoint != "" {
					mountPoint = vol.MountPoint
					break
				}
			}
		}

		// Find APFS volume device (e.g. disk5s1) for this physical disk's partitions
		volumeDevice := ""
		for _, part := range diskInfo.Partitions {
			if v, ok := apfsVolumeMap[part.DeviceIdentifier]; ok {
				volumeDevice = v
				break
			}
		}

		drive := Drive{
			Name:         detailedInfo.MediaName,
			Path:         fmt.Sprintf("/dev/%s", diskInfo.DeviceIdentifier),
			MountPoint:   mountPoint,
			VolumeDevice: volumeDevice,
			Size:         detailedInfo.Size,
			Type:         determineDriveType(detailedInfo),
			IsRemovable:  detailedInfo.Removable || detailedInfo.Ejectable,
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

// findAPFSVolumeDevice returns the APFS volume device (e.g. "disk5s1") for a physical disk path.
func (m *Manager) findAPFSVolumeDevice(physicalDiskPath string) (string, error) {
	diskID := strings.TrimPrefix(physicalDiskPath, "/dev/")

	cmd := exec.Command("diskutil", "list", "-plist")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run diskutil list: %w", err)
	}

	var diskList DiskUtilList
	decoder := plist.NewDecoder(bytes.NewReader(output))
	if err := decoder.Decode(&diskList); err != nil {
		return "", fmt.Errorf("failed to parse diskutil plist: %w", err)
	}

	for _, diskInfo := range diskList.AllDisksAndPartitions {
		if !strings.HasPrefix(diskInfo.Content, "Apple_APFS") {
			continue
		}
		for _, store := range diskInfo.APFSPhysicalStores {
			storeID := store.DeviceIdentifier
			if storeID == diskID || strings.HasPrefix(storeID, diskID+"s") {
				for _, vol := range diskInfo.APFSVolumes {
					if vol.DeviceIdentifier != "" {
						return vol.DeviceIdentifier, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("no APFS volume found for %s — drive may not be APFS formatted", physicalDiskPath)
}

// encryptDarwin enables macOS native FileVault encryption on the APFS volume. Non-destructive.
func (m *Manager) encryptDarwin(d *Drive, password []byte) error {
	volumeDevice := d.VolumeDevice
	if volumeDevice == "" {
		var err error
		volumeDevice, err = m.findAPFSVolumeDevice(d.Path)
		if err != nil {
			return fmt.Errorf(
				"drive is not APFS formatted — encryption requires APFS\n" +
					"  Fix: open Disk Utility → select the drive → Erase → choose APFS format\n" +
					"  (This erases the drive once. After that, all data stored on it is encrypted.)",
			)
		}
	}

	// diskutil reads passphrase from stdin when not connected to a terminal.
	// Send it twice (passphrase + confirm).
	passphraseInput := string(password) + "\n" + string(password) + "\n"
	cmd := exec.Command("diskutil", "apfs", "encryptVolume", volumeDevice, "-user", "disk")
	cmd.Stdin = strings.NewReader(passphraseInput)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable FileVault: %s\n(hint: try running with sudo)", strings.TrimSpace(string(output)))
	}
	return nil
}

// decryptDarwin removes FileVault encryption from the APFS volume. Non-destructive.
func (m *Manager) decryptDarwin(d *Drive, password []byte) error {
	volumeDevice := d.VolumeDevice
	if volumeDevice == "" {
		var err error
		volumeDevice, err = m.findAPFSVolumeDevice(d.Path)
		if err != nil {
			return fmt.Errorf("could not find APFS volume for %s: %w", d.Path, err)
		}
	}

	output, err := runCommand("diskutil", "apfs", "decryptVolume", volumeDevice, "-passphrase", string(password))
	if err != nil {
		return fmt.Errorf("decryption failed: %s", strings.TrimSpace(output))
	}
	return nil
}

// unlockDarwin mounts an APFS encrypted volume using the given passphrase.
func (m *Manager) unlockDarwin(d *Drive, password []byte) error {
	volumeDevice := d.VolumeDevice
	if volumeDevice == "" {
		var err error
		volumeDevice, err = m.findAPFSVolumeDevice(d.Path)
		if err != nil {
			return fmt.Errorf("could not find APFS volume for %s: %w", d.Path, err)
		}
	}

	output, err := runCommand("diskutil", "apfs", "unlockVolume", volumeDevice, "-passphrase", string(password))
	if err != nil {
		return fmt.Errorf("unlock failed: %s", strings.TrimSpace(output))
	}
	return nil
}

// lockDarwin unmounts the drive's volume.
func (m *Manager) lockDarwin(d *Drive) error {
	target := d.MountPoint
	if target == "" {
		target = d.Path
	}
	output, err := runCommand("diskutil", "unmount", target)
	if err != nil {
		return fmt.Errorf("lock failed: %s", strings.TrimSpace(output))
	}
	return nil
}

// checkStatusDarwin checks encryption status via diskutil (no root required for APFS).
// Falls back to reading the custom encrypto-pro header if APFS reports not encrypted.
func (m *Manager) checkStatusDarwin(drivePath string) (*Status, error) {
	status := &Status{DevicePath: drivePath}

	diskID := strings.TrimSuffix(drivePath, "/")
	if strings.HasPrefix(diskID, "/dev/") {
		diskID = strings.TrimPrefix(diskID, "/dev/")
	}

	info, err := m.getDiskInfo(diskID)
	if err != nil {
		status.Error = fmt.Sprintf("cannot get disk info: %v", err)
		return status, nil
	}

	status.IsMounted = info.MountPoint != ""

	if info.Encrypted || info.FileVault {
		status.IsEncrypted = true
		status.Method = "apfs"
		return status, nil
	}

	// Physical disk doesn't show encryption — check if this disk has APFS volumes
	// that might be encrypted (encryption status lives on the volume, not the physical disk)
	volumeDevice, volErr := m.findAPFSVolumeDevice(drivePath)
	if volErr == nil && volumeDevice != "" {
		volInfo, volErr := m.getDiskInfo(volumeDevice)
		if volErr == nil && (volInfo.Encrypted || volInfo.FileVault) {
			status.IsEncrypted = true
			status.Method = "apfs"
			status.DevicePath = "/dev/" + volumeDevice
			return status, nil
		}
	}

	// Not APFS encrypted — check for a custom encrypto-pro header on the raw device.
	// This requires root; if it fails we simply report not encrypted.
	handle, err := openDarwinDrive(drivePath)
	if err != nil {
		return status, nil
	}
	defer handle.Close()

	buf := make([]byte, 512)
	if _, err := handle.Read(0, buf); err != nil {
		return status, nil
	}
	var header Header
	header.Decode(buf)
	if string(header.Magic[:]) == HeaderMagic {
		status.IsEncrypted = true
		status.Method = "raw"
		status.Version = header.Version
		status.HasHidden = header.HasHidden
	}
	return status, nil
}

// resolveDarwinPath converts any path (mount point, volume path, /dev/ path) to /dev/diskX.
func (m *Manager) resolveDarwinPath(path string) string {
	// Already a device path
	if strings.HasPrefix(path, "/dev/") {
		return path
	}

	info, err := m.getDiskInfo(strings.TrimSuffix(path, "/"))
	if err != nil {
		return path
	}

	if info.WholeDisk {
		return "/dev/" + info.DeviceIdentifier
	}
	if info.ParentWholeDisk != "" {
		return "/dev/" + info.ParentWholeDisk
	}
	return "/dev/" + info.DeviceIdentifier
}
