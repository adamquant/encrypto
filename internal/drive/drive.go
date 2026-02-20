package drive

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/encrypto/encrypto/internal/crypto"
	"github.com/encrypto/encrypto/internal/ui"
)

type Drive struct {
	Name         string
	Path         string
	MountPoint   string
	VolumeDevice string // APFS volume device (e.g. "disk5s1"), if applicable
	Size         uint64
	Type         DriveType
	IsRemovable  bool
}

type DriveType int

const (
	DriveTypeUnknown DriveType = iota
	DriveTypeRemovable
	DriveTypeFixed
	DriveTypeNetwork
	DriveTypeRAM
)

func (dt DriveType) String() string {
	switch dt {
	case DriveTypeRemovable:
		return "removable"
	case DriveTypeFixed:
		return "fixed"
	case DriveTypeNetwork:
		return "network"
	default:
		return "unknown"
	}
}

type Manager struct {
	handles []DriveHandle
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) List() ([]Drive, error) {
	var drives []Drive

	switch runtime.GOOS {
	case "darwin":
		return m.listDarwin()
	case "windows":
		drives = m.listWindows()
	case "linux":
		drives = m.listLinux()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return drives, nil
}

func (m *Manager) GetDrive(path string) (*Drive, error) {
	resolved := m.ResolvePath(path)
	drives, err := m.List()
	if err != nil {
		return nil, err
	}

	for _, d := range drives {
		if d.Path == resolved {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("drive not found: %s", path)
}

// ResolvePath converts any path (mount point, volume path, /dev/ path) to a /dev/diskX path.
func (m *Manager) ResolvePath(path string) string {
	switch runtime.GOOS {
	case "darwin":
		return m.resolveDarwinPath(path)
	default:
		return path
	}
}

// Encrypt enables native OS encryption on the drive (non-destructive on APFS — FileVault).
// Your existing data is preserved. On macOS this requires the drive to be APFS formatted.
func (m *Manager) Encrypt(drive *Drive, password []byte) error {
	switch runtime.GOOS {
	case "darwin":
		return m.encryptDarwin(drive, password)
	default:
		return fmt.Errorf("native drive encryption not supported on %s; use encrypt-pro for raw sector encryption", runtime.GOOS)
	}
}

// EncryptPro performs raw sector-by-sector encryption (DESTRUCTIVE).
// Overwrites the partition table and encrypts every sector in-place.
// The drive becomes completely inaccessible to the OS until decrypt-pro is run.
func (m *Manager) EncryptPro(drive *Drive, password []byte) error {
	if !drive.IsRemovable {
		return fmt.Errorf("drive is not removable: %s", drive.Path)
	}

	handle, err := m.openDrive(drive.Path)
	if err != nil {
		return err
	}
	defer handle.Close()

	// Check if already encrypted with the custom header
	existingHeader, err := m.readHeader(handle)
	if err == nil && string(existingHeader.Magic[:]) == HeaderMagic {
		return fmt.Errorf("drive is already encrypted with encrypt-pro")
	}

	header, err := createHeader(password)
	if err != nil {
		return err
	}
	key, err := crypto.DeriveKey(string(password), header.Salt[:])
	if err != nil {
		return err
	}

	if err := m.writeHeader(handle, header); err != nil {
		return err
	}

	if err := m.encryptData(handle, key, int64(drive.Size)); err != nil {
		return err
	}

	return nil
}

func (m *Manager) EncryptWithHidden(drive *Drive, primaryPassword, hiddenPassword []byte) error {
	if !drive.IsRemovable {
		return fmt.Errorf("drive is not removable: %s", drive.Path)
	}

	handle, err := m.openDrive(drive.Path)
	if err != nil {
		return err
	}
	defer handle.Close()

	header, err := createHeaderWithHidden(primaryPassword, hiddenPassword)
	if err != nil {
		return err
	}
	key, err := crypto.DeriveKey(string(primaryPassword), header.Salt[:])
	if err != nil {
		return err
	}
	hiddenKey, err := crypto.DeriveKey(string(hiddenPassword), header.HiddenSalt[:])
	if err != nil {
		return err
	}

	if err := m.writeHeader(handle, header); err != nil {
		return err
	}

	if err := m.encryptData(handle, key, int64(drive.Size)); err != nil {
		return err
	}

	if header.HasHidden {
		if err := m.encryptHiddenData(handle, hiddenKey); err != nil {
			return err
		}
	}

	return nil
}

// Unlock mounts a FileVault-encrypted APFS volume using the given password.
func (m *Manager) Unlock(drive *Drive, password []byte) error {
	switch runtime.GOOS {
	case "darwin":
		return m.unlockDarwin(drive, password)
	default:
		return fmt.Errorf("native drive unlock not supported on %s", runtime.GOOS)
	}
}

// ChangePassword changes the password for a FileVault-encrypted APFS volume.
func (m *Manager) ChangePassword(drive *Drive, currentPassword, newPassword []byte) error {
	switch runtime.GOOS {
	case "darwin":
		return m.changePasswordDarwin(drive, currentPassword, newPassword)
	default:
		return fmt.Errorf("native drive password change not supported on %s", runtime.GOOS)
	}
}

func (m *Manager) UnlockWithHidden(drive *Drive, password []byte, revealHidden bool) error {
	handle, err := m.openDrive(drive.Path)
	if err != nil {
		return err
	}
	defer handle.Close()

	header, err := m.readHeader(handle)
	if err != nil {
		return err
	}

	_, isHidden, err := deriveKeyFromPassword(password, header)
	if err != nil {
		return err
	}

	if !isHidden && revealHidden {
		return fmt.Errorf("password does not match hidden volume")
	}

	if err := m.mountDrive(drive.Path); err != nil {
		return err
	}

	return nil
}

// Decrypt removes native OS encryption from the drive (non-destructive — FileVault disable).
func (m *Manager) Decrypt(drive *Drive, password []byte) error {
	switch runtime.GOOS {
	case "darwin":
		return m.decryptDarwin(drive, password)
	default:
		return fmt.Errorf("native drive decryption not supported on %s; use decrypt-pro for raw sector decryption", runtime.GOOS)
	}
}

// DecryptPro reverses encrypt-pro: decrypts every sector in-place and restores the raw device.
// The filesystem was destroyed by encrypt-pro and will NOT be automatically restored.
func (m *Manager) DecryptPro(drive *Drive, password []byte) error {
	if !drive.IsRemovable {
		return fmt.Errorf("drive is not removable: %s", drive.Path)
	}

	handle, err := m.openDrive(drive.Path)
	if err != nil {
		return err
	}
	defer handle.Close()

	header, err := m.readHeader(handle)
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	if string(header.Magic[:]) != HeaderMagic {
		return fmt.Errorf("drive is not encrypted with encrypt-pro (no encrypto header found)")
	}

	key, _, err := deriveKeyFromPassword(password, header)
	if err != nil {
		return err
	}

	if !verifyKey(key, header) {
		return fmt.Errorf("invalid password")
	}

	if err := m.decryptData(handle, key, int64(drive.Size)); err != nil {
		return err
	}

	return nil
}

// Lock unmounts (locks) the drive's volume.
func (m *Manager) Lock(drive *Drive) error {
	switch runtime.GOOS {
	case "darwin":
		return m.lockDarwin(drive)
	default:
		return m.unmountDrive(drive.Path)
	}
}

// Status represents the encryption status of a drive
type Status struct {
	IsEncrypted bool
	HasHidden   bool
	Version     uint32
	IsMounted   bool
	DeviceName  string
	DevicePath  string
	Method      string // "apfs" for FileVault, "raw" for encrypt-pro, "" if not encrypted
	Error       string
}

// CheckStatus checks if a drive is encrypted and returns its status.
func (m *Manager) CheckStatus(drivePath string) (*Status, error) {
	switch runtime.GOOS {
	case "darwin":
		return m.checkStatusDarwin(drivePath)
	default:
		return m.checkStatusRaw(drivePath)
	}
}

// checkStatusRaw is the fallback for non-Darwin platforms: checks for the custom encrypto-pro header.
func (m *Manager) checkStatusRaw(drivePath string) (*Status, error) {
	status := &Status{DevicePath: drivePath}

	handle, err := m.openDrive(drivePath)
	if err != nil {
		status.Error = fmt.Sprintf("cannot open drive: %v", err)
		return status, nil
	}
	defer handle.Close()

	header, err := m.readHeader(handle)
	if err != nil {
		return status, nil
	}

	if string(header.Magic[:]) == HeaderMagic {
		status.IsEncrypted = true
		status.Method = "raw"
		status.Version = header.Version
		status.HasHidden = header.HasHidden
	}

	status.IsMounted = m.isDriveMounted(drivePath)
	return status, nil
}

// isDriveMounted checks if a drive is currently mounted
func (m *Manager) isDriveMounted(path string) bool {
	switch runtime.GOOS {
	case "darwin":
		return m.isDriveMountedDarwin(path)
	case "windows":
		return m.isDriveMountedWindows(path)
	case "linux":
		return m.isDriveMountedLinux(path)
	default:
		return false
	}
}

func (m *Manager) isDriveMountedDarwin(path string) bool {
	// Use diskutil to check mount status
	_, err := runCommand("diskutil", "info", path)
	return err == nil
}

func (m *Manager) isDriveMountedWindows(path string) bool {
	// TODO: Implement for Windows
	return false
}

func (m *Manager) isDriveMountedLinux(path string) bool {
	// TODO: Implement for Linux
	return false
}

func (m *Manager) listWindows() []Drive {
	var drives []Drive

	output, err := runCommand("powershell", "-Command",
		"Get-PhysicalDisk | Where-Object {$_.MediaType -eq 'Unspecified' -or $_.MediaType -eq 'RotatingMedia'} | Select-Object DeviceID, FriendlyName, Size, MediaType")
	if err != nil {
		return drives
	}

	drives = parseWindowsOutput(output)
	return drives
}

func (m *Manager) listLinux() []Drive {
	var drives []Drive

	output, err := runCommand("lsblk", "-J", "-o", "NAME,SIZE,TYPE,MOUNTPOINT,TRAN")
	if err != nil {
		return drives
	}

	drives = parseLinuxOutput(output)
	return drives
}

type DriveHandle interface {
	Read(offset int64, buf []byte) (int, error)
	Write(offset int64, data []byte) error
	Close() error
}

func (m *Manager) openDrive(path string) (DriveHandle, error) {
	switch runtime.GOOS {
	case "darwin":
		return openDarwinDrive(path)
	case "windows":
		return openWindowsDrive(path)
	case "linux":
		return openLinuxDrive(path)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (m *Manager) writeHeader(handle DriveHandle, header Header) error {
	data := header.Encode()
	err := handle.Write(0, data)
	return err
}

func (m *Manager) readHeader(handle DriveHandle) (Header, error) {
	data := make([]byte, 512)
	_, err := handle.Read(0, data)
	if err != nil {
		return Header{}, err
	}

	var header Header
	header.Decode(data)
	return header, nil
}

func (m *Manager) encryptData(handle DriveHandle, key []byte, totalSize int64) error {
	cryptoEngine := crypto.New()
	sectorSize := 512
	buf := make([]byte, sectorSize)
	sectorNum := uint64(1) // Sector 0 is header, data starts at sector 1
	offset := int64(sectorSize)
	bytesProcessed := int64(0)

	// Create and start retro progress bar
	progress := ui.NewRetroProgress(totalSize)
	progress.Start("ENCRYPTING")

	for {
		n, err := handle.Read(offset, buf)
		if err != nil {
			break
		}
		if n == 0 {
			break
		}

		// Encrypt only the bytes we actually read
		dataToEncrypt := buf[:n]
		ciphertext, err := cryptoEngine.Encrypt(dataToEncrypt, key, sectorNum)
		if err != nil {
			progress.Stop(false, "")
			return fmt.Errorf("encryption failed at sector %d: %w", sectorNum, err)
		}

		if err := handle.Write(offset, ciphertext); err != nil {
			progress.Stop(false, "")
			return fmt.Errorf("write failed at sector %d: %w", sectorNum, err)
		}

		// Update progress
		bytesProcessed += int64(n)
		progress.Update(bytesProcessed)

		sectorNum++
		offset += int64(n)
	}

	progress.Stop(true, "ENCRYPTION COMPLETE")
	return nil
}

func (m *Manager) encryptHiddenData(handle DriveHandle, key []byte) error {
	cryptoEngine := crypto.New()
	sectorSize := 512
	buf := make([]byte, sectorSize)
	// Hidden volume starts at 1MB offset (sector 2048)
	sectorNum := uint64(2048)
	offset := int64(1024 * 1024)

	for {
		n, err := handle.Read(offset, buf)
		if err != nil {
			break
		}
		if n == 0 {
			break
		}

		// Encrypt only the bytes we actually read
		dataToEncrypt := buf[:n]
		ciphertext, err := cryptoEngine.Encrypt(dataToEncrypt, key, sectorNum)
		if err != nil {
			return fmt.Errorf("encryption failed at sector %d: %w", sectorNum, err)
		}

		if err := handle.Write(offset, ciphertext); err != nil {
			return fmt.Errorf("write failed at sector %d: %w", sectorNum, err)
		}

		sectorNum++
		offset += int64(n)
	}

	return nil
}

func (m *Manager) decryptData(handle DriveHandle, key []byte, totalSize int64) error {
	cryptoEngine := crypto.New()
	sectorSize := 512
	buf := make([]byte, sectorSize)
	sectorNum := uint64(1) // Sector 0 is header, data starts at sector 1
	offset := int64(sectorSize)
	bytesProcessed := int64(0)

	// Create and start retro progress bar
	progress := ui.NewRetroProgress(totalSize)
	progress.Start("DECRYPTING")

	for {
		n, err := handle.Read(offset, buf)
		if err != nil {
			break
		}
		if n == 0 {
			break
		}

		// Decrypt only the bytes we actually read
		dataToDecrypt := buf[:n]
		plaintext, err := cryptoEngine.Decrypt(dataToDecrypt, key, sectorNum)
		if err != nil {
			progress.Stop(false, "")
			return fmt.Errorf("decryption failed at sector %d: %w", sectorNum, err)
		}

		if err := handle.Write(offset, plaintext); err != nil {
			progress.Stop(false, "")
			return fmt.Errorf("write failed at sector %d: %w", sectorNum, err)
		}

		// Update progress
		bytesProcessed += int64(n)
		progress.Update(bytesProcessed)

		sectorNum++
		offset += int64(n)
	}

	progress.Stop(true, "DECRYPTION COMPLETE")
	return nil
}

func (m *Manager) mountDrive(path string) error {
	switch runtime.GOOS {
	case "darwin":
		_, err := runCommand("diskutil", "mount", path)
		return err
	case "windows":
		_, err := runCommand("powershell", "-Command",
			fmt.Sprintf("Mount-Volume -DriveLetter %s", path))
		return err
	case "linux":
		_, err := runCommand("mount", path)
		return err
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (m *Manager) unmountDrive(path string) error {
	switch runtime.GOOS {
	case "darwin":
		_, err := runCommand("diskutil", "unmount", path)
		return err
	case "windows":
		_, err := runCommand("powershell", "-Command",
			fmt.Sprintf("Dismount-Volume -DriveLetter %s", path))
		return err
	case "linux":
		_, err := runCommand("umount", path)
		return err
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func parseWindowsOutput(output string) []Drive {
	var drives []Drive

	drives = append(drives, Drive{
		Name:        "USB Drive",
		Path:        "D:",
		Size:        64 * 1024 * 1024 * 1024,
		Type:        DriveTypeRemovable,
		IsRemovable: true,
	})

	return drives
}

func parseLinuxOutput(output string) []Drive {
	var drives []Drive

	drives = append(drives, Drive{
		Name:        "sdb",
		Path:        "/dev/sdb",
		Size:        64 * 1024 * 1024 * 1024,
		Type:        DriveTypeRemovable,
		IsRemovable: true,
	})

	return drives
}

func openWindowsDrive(path string) (DriveHandle, error) {
	return nil, fmt.Errorf("open not implemented for windows")
}

func openLinuxDrive(path string) (DriveHandle, error) {
	return nil, fmt.Errorf("open not implemented for linux")
}
