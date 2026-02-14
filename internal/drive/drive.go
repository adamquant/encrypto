package drive

import (
	"fmt"
	"runtime"

	"github.com/encrypto/encrypto/internal/crypto"
)

type Drive struct {
	Name        string
	Path        string
	Size        uint64
	Type        DriveType
	IsRemovable bool
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
	drives, err := m.List()
	if err != nil {
		return nil, err
	}

	for _, d := range drives {
		if d.Path == path {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("drive not found: %s", path)
}

func (m *Manager) Encrypt(drive *Drive, password []byte) error {
	if !drive.IsRemovable {
		return fmt.Errorf("drive is not removable: %s", drive.Path)
	}

	handle, err := m.openDrive(drive.Path)
	if err != nil {
		return err
	}
	defer handle.Close()

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

	if err := m.encryptData(handle, key); err != nil {
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

	if err := m.encryptData(handle, key); err != nil {
		return err
	}

	if header.HasHidden {
		if err := m.encryptHiddenData(handle, hiddenKey); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Unlock(drive *Drive, password []byte) error {
	handle, err := m.openDrive(drive.Path)
	if err != nil {
		return err
	}
	defer handle.Close()

	header, err := m.readHeader(handle)
	if err != nil {
		return err
	}

	key, _, err := deriveKeyFromPassword(password, header)
	if err != nil {
		return err
	}

	if !verifyKey(key, header) {
		return fmt.Errorf("invalid password")
	}

	if err := m.mountDrive(drive.Path); err != nil {
		return err
	}

	return nil
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

func (m *Manager) Lock(drive *Drive) error {
	if err := m.unmountDrive(drive.Path); err != nil {
		return err
	}

	return nil
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

func (m *Manager) encryptData(handle DriveHandle, key []byte) error {
	buf := make([]byte, 4096)
	offset := int64(512)

	for {
		n, err := handle.Read(offset, buf)
		if err != nil {
			break
		}
		if n == 0 {
			break
		}

		ciphertext := make([]byte, n)
		copy(ciphertext, buf[:n])

		if err := handle.Write(offset, ciphertext); err != nil {
			return err
		}

		offset += int64(n)
	}

	return nil
}

func (m *Manager) encryptHiddenData(handle DriveHandle, key []byte) error {
	buf := make([]byte, 4096)
	offset := int64(1024 * 1024)

	for {
		n, err := handle.Read(offset, buf)
		if err != nil {
			break
		}
		if n == 0 {
			break
		}

		ciphertext := make([]byte, n)
		copy(ciphertext, buf[:n])

		if err := handle.Write(offset, ciphertext); err != nil {
			return err
		}

		offset += int64(n)
	}

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
	return "", nil
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

func openDarwinDrive(path string) (DriveHandle, error) {
	return nil, fmt.Errorf("open not implemented for darwin")
}

func openWindowsDrive(path string) (DriveHandle, error) {
	return nil, fmt.Errorf("open not implemented for windows")
}

func openLinuxDrive(path string) (DriveHandle, error) {
	return nil, fmt.Errorf("open not implemented for linux")
}
