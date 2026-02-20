package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/encrypto/encrypto/internal/crypto"
	"github.com/encrypto/encrypto/internal/drive"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "encrypt":
		handleEncrypt(os.Args[2:])
	case "decrypt":
		handleDecrypt(os.Args[2:])
	case "encrypt-pro":
		handleEncryptPro(os.Args[2:])
	case "decrypt-pro":
		handleDecryptPro(os.Args[2:])
	case "encrypt-file":
		handleEncryptFile(os.Args[2:])
	case "decrypt-file":
		handleDecryptFile(os.Args[2:])
	case "unlock":
		handleUnlock(os.Args[2:])
	case "lock":
		handleLock(os.Args[2:])
	case "change-password":
		handleChangePassword(os.Args[2:])
	case "unlock-pro":
		handleUnlockPro(os.Args[2:])
	case "lock-pro":
		handleLockPro(os.Args[2:])
	case "change-password-pro":
		handleChangePasswordPro(os.Args[2:])
	case "status":
		handleStatus(os.Args[2:])
	case "list":
		handleList()
	case "size":
		handleSize(os.Args[2:])
	case "encrypt-hidden":
		handleEncryptHidden(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("encrypto - Drive & file encryption")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  encrypto encrypt <drive>               Enable FileVault on APFS drive (safe, non-destructive)")
	fmt.Println("  encrypto decrypt <drive>               Remove FileVault from APFS drive")
	fmt.Println("  encrypto unlock <drive>                Mount an encrypted APFS drive")
	fmt.Println("  encrypto lock <drive>                  Unmount a drive")
	fmt.Println("  encrypto change-password <drive>       Change FileVault password")
	fmt.Println("  encrypto status <drive>                Check encryption status")
	fmt.Println("  encrypto list                          List available drives")
	fmt.Println("  encrypto size <path>                   Show size of drive, volume, or folder")
	fmt.Println()
	fmt.Println("  encrypt-file <file> [output]           Encrypt a file (instant)")
	fmt.Println("    1 arg:  Encrypt in-place (replaces original)")
	fmt.Println("    2 args: Create new encrypted file")
	fmt.Println("  decrypt-file <file> [output]           Decrypt a file (instant)")
	fmt.Println("    1 arg:  Decrypt in-place (replaces encrypted file)")
	fmt.Println("    2 args: Create new decrypted file")
	fmt.Println()
	fmt.Println("  [ADVANCED] Raw sector encryption — cross-platform, works on Mac/Windows/Linux:")
	fmt.Println("  encrypto encrypt-pro <drive>           Raw sector encrypt (wipes partition table)")
	fmt.Println("  encrypto decrypt-pro <drive>          Raw sector decrypt (restore raw device)")
	fmt.Println("  encrypto unlock-pro <drive>           Mount an encrypt-pro encrypted drive")
	fmt.Println("  encrypto lock-pro <drive>             Unmount an encrypt-pro encrypted drive")
	fmt.Println("  encrypto change-password-pro <drive>  Change encrypt-pro password (re-encrypts all data)")
	fmt.Println("  encrypto encrypt-hidden <drive>       Raw sector encrypt with hidden volume")
}

func handleList() {
	manager := drive.NewManager()
	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	if len(drives) == 0 {
		fmt.Println("No drives found")
		return
	}

	fmt.Println("Available drives:")
	fmt.Println()
	for _, d := range drives {
		fmt.Printf("  %s (%s) - %d GB %s\n",
			d.Path, d.Name, d.Size/(1024*1024*1024),
			d.Type)
		if d.MountPoint != "" {
			fmt.Printf("    mounted at: %s\n", d.MountPoint)
		}
		if d.IsRemovable {
			fmt.Println("    [removable]")
		}
	}
}

func handleEncrypt(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto encrypt <drive-path>")
		os.Exit(1)
	}

	manager := drive.NewManager()
	drivePath := manager.ResolvePath(args[0])

	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	var selectedDrive *drive.Drive
	for _, d := range drives {
		if d.Path == drivePath {
			selectedDrive = &d
			break
		}
	}

	if selectedDrive == nil {
		fmt.Printf("Drive not found: %s\n", drivePath)
		os.Exit(1)
	}

	if !selectedDrive.IsRemovable {
		fmt.Printf("Error: %s is not a removable drive\n", drivePath)
		os.Exit(1)
	}

	fmt.Printf("Enabling FileVault encryption on %s (%s)\n", selectedDrive.Name, selectedDrive.Path)
	fmt.Println()
	fmt.Println("Your existing data will NOT be deleted.")
	fmt.Println("Encryption runs in the background — the drive stays usable immediately.")
	fmt.Println("After this, the drive will require your password to mount.")
	fmt.Println()
	fmt.Println("Make sure you remember your password. There is no recovery option.")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nContinue? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "yes" {
		fmt.Println("Cancelled")
		return
	}

	fmt.Print("Enter password: ")
	password1, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Confirm password: ")
	password2, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	if string(password1) != string(password2) {
		fmt.Println("Passwords do not match")
		os.Exit(1)
	}

	err = manager.Encrypt(selectedDrive, password1)
	if err != nil {
		fmt.Printf("Encryption failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("FileVault encryption enabled. The drive is encrypting in the background.")
}

func handleDecrypt(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto decrypt <drive-path>")
		os.Exit(1)
	}

	manager := drive.NewManager()
	drivePath := manager.ResolvePath(args[0])

	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	var selectedDrive *drive.Drive
	for _, d := range drives {
		if d.Path == drivePath {
			selectedDrive = &d
			break
		}
	}

	if selectedDrive == nil {
		fmt.Printf("Drive not found: %s\n", drivePath)
		os.Exit(1)
	}

	fmt.Printf("Removing FileVault encryption from %s (%s)\n", selectedDrive.Name, selectedDrive.Path)
	fmt.Println("Your data will remain intact. Decryption runs in the background.")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Continue? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "yes" {
		fmt.Println("Cancelled")
		return
	}

	fmt.Print("Enter password: ")
	password, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	err = manager.Decrypt(selectedDrive, password)
	if err != nil {
		fmt.Printf("Decryption failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("FileVault encryption removed. The drive is decrypting in the background.")
}

func handleEncryptPro(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto encrypt-pro <drive-path>")
		os.Exit(1)
	}

	manager := drive.NewManager()
	drivePath := manager.ResolvePath(args[0])

	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	var selectedDrive *drive.Drive
	for _, d := range drives {
		if d.Path == drivePath {
			selectedDrive = &d
			break
		}
	}

	if selectedDrive == nil {
		fmt.Printf("Drive not found: %s\n", drivePath)
		os.Exit(1)
	}

	if !selectedDrive.IsRemovable {
		fmt.Printf("Error: %s is not a removable drive\n", drivePath)
		os.Exit(1)
	}

	fmt.Println("████████████████████████████████████████████████████████")
	fmt.Println("  WARNING: ENCRYPT-PRO — RAW SECTOR ENCRYPTION")
	fmt.Println()
	fmt.Printf("  Target: %s (%s)\n", selectedDrive.Name, selectedDrive.Path)
	fmt.Println()
	fmt.Println("  THIS WILL PERMANENTLY DESTROY:")
	fmt.Println("    - The partition table (sector 0 overwritten with header)")
	fmt.Println("    - All file data (every sector encrypted in-place)")
	fmt.Println()
	fmt.Println("  After this, the drive CANNOT be mounted by any OS.")
	fmt.Println("  You must run decrypt-pro to restore the raw device.")
	fmt.Println("  The filesystem will NOT be recovered — it is gone.")
	fmt.Println()
	fmt.Println("  BACK UP ALL DATA BEFORE PROCEEDING.")
	fmt.Println("████████████████████████████████████████████████████████")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nType YES in capitals to confirm: ")
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(confirm) != "YES" {
		fmt.Println("Cancelled")
		return
	}

	fmt.Print("Enter password: ")
	password1, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Confirm password: ")
	password2, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	if string(password1) != string(password2) {
		fmt.Println("Passwords do not match")
		os.Exit(1)
	}

	err = manager.EncryptPro(selectedDrive, password1)
	if err != nil {
		fmt.Printf("Encryption failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Raw sector encryption complete.")
}

func handleDecryptPro(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto decrypt-pro <drive-path>")
		os.Exit(1)
	}

	manager := drive.NewManager()
	drivePath := manager.ResolvePath(args[0])

	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	var selectedDrive *drive.Drive
	for _, d := range drives {
		if d.Path == drivePath {
			selectedDrive = &d
			break
		}
	}

	if selectedDrive == nil {
		fmt.Printf("Drive not found: %s\n", drivePath)
		os.Exit(1)
	}

	fmt.Println("████████████████████████████████████████████████████████")
	fmt.Println("  DECRYPT-PRO — Raw sector decryption")
	fmt.Printf("  Target: %s (%s)\n", selectedDrive.Name, selectedDrive.Path)
	fmt.Println("  Note: filesystem was destroyed by encrypt-pro and will")
	fmt.Println("  not be automatically restored after this operation.")
	fmt.Println("████████████████████████████████████████████████████████")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nContinue? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "yes" {
		fmt.Println("Cancelled")
		return
	}

	fmt.Print("Enter password: ")
	password, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	err = manager.DecryptPro(selectedDrive, password)
	if err != nil {
		fmt.Printf("Decryption failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Raw sector decryption complete.")
}

func handleEncryptFile(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto encrypt-file <input-file> [output-file]")
		fmt.Println("  1 arg:  In-place encryption (replaces original file)")
		fmt.Println("  2 args: Create new encrypted file")
		os.Exit(1)
	}

	inputPath := args[0]
	inPlace := len(args) == 1
	var outputPath string

	if inPlace {
		outputPath = inputPath + ".tmp.enc"
	} else {
		outputPath = args[1]
	}

	fmt.Printf("Encrypting file: %s\n", inputPath)
	fmt.Print("Enter password: ")
	password, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Confirm password: ")
	password2, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	if string(password) != string(password2) {
		fmt.Println("Passwords do not match")
		os.Exit(1)
	}

	err = crypto.EncryptFile(inputPath, outputPath, password)
	if err != nil {
		fmt.Printf("Encryption failed: %v\n", err)
		os.Exit(1)
	}

	if inPlace {
		// Replace original with encrypted version
		if err := os.Remove(inputPath); err != nil {
			fmt.Printf("Warning: Could not remove original file: %v\n", err)
		}
		if err := os.Rename(outputPath, inputPath); err != nil {
			fmt.Printf("Error: Could not rename encrypted file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ File encrypted in-place: %s\n", inputPath)
	} else {
		fmt.Printf("✓ File encrypted: %s\n", outputPath)
	}
}

func handleDecryptFile(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto decrypt-file <input-file> [output-file]")
		fmt.Println("  1 arg:  In-place decryption (replaces encrypted file)")
		fmt.Println("  2 args: Create new decrypted file")
		os.Exit(1)
	}

	inputPath := args[0]
	inPlace := len(args) == 1
	var outputPath string

	if inPlace {
		outputPath = inputPath + ".tmp.dec"
	} else {
		outputPath = args[1]
	}

	fmt.Printf("Decrypting file: %s\n", inputPath)
	fmt.Print("Enter password: ")
	password, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	err = crypto.DecryptFile(inputPath, outputPath, password)
	if err != nil {
		fmt.Printf("Decryption failed: %v\n", err)
		os.Exit(1)
	}

	if inPlace {
		// Replace encrypted with decrypted version
		if err := os.Remove(inputPath); err != nil {
			fmt.Printf("Warning: Could not remove encrypted file: %v\n", err)
		}
		if err := os.Rename(outputPath, inputPath); err != nil {
			fmt.Printf("Error: Could not rename decrypted file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ File decrypted in-place: %s\n", inputPath)
	} else {
		fmt.Printf("✓ File decrypted: %s\n", outputPath)
	}
}

func handleLock(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto lock <drive-path>")
		os.Exit(1)
	}

	manager := drive.NewManager()
	drivePath := manager.ResolvePath(args[0])

	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	var selectedDrive *drive.Drive
	for _, d := range drives {
		if d.Path == drivePath {
			selectedDrive = &d
			break
		}
	}

	if selectedDrive == nil {
		fmt.Printf("Drive not found: %s\n", drivePath)
		os.Exit(1)
	}

	fmt.Printf("Locking %s (%s)...\n", selectedDrive.Name, selectedDrive.Path)

	err = manager.Lock(selectedDrive)
	if err != nil {
		fmt.Printf("Lock failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Drive locked successfully")
}

func handleUnlock(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto unlock <drive-path>")
		os.Exit(1)
	}

	manager := drive.NewManager()
	drivePath := manager.ResolvePath(args[0])

	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	var selectedDrive *drive.Drive
	for _, d := range drives {
		if d.Path == drivePath {
			selectedDrive = &d
			break
		}
	}

	if selectedDrive == nil {
		fmt.Printf("Drive not found: %s\n", drivePath)
		os.Exit(1)
	}

	fmt.Printf("Unlocking %s (%s)...\n", selectedDrive.Name, selectedDrive.Path)

	fmt.Print("Enter password: ")
	password, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	err = manager.Unlock(selectedDrive, password)
	if err != nil {
		fmt.Printf("Unlock failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Drive unlocked successfully")
}

func handleChangePassword(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto change-password <drive-path>")
		os.Exit(1)
	}

	manager := drive.NewManager()
	drivePath := manager.ResolvePath(args[0])

	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	var selectedDrive *drive.Drive
	for _, d := range drives {
		if d.Path == drivePath {
			selectedDrive = &d
			break
		}
	}

	if selectedDrive == nil {
		fmt.Printf("Drive not found: %s\n", drivePath)
		os.Exit(1)
	}

	fmt.Printf("Changing password on %s (%s)...\n", selectedDrive.Name, selectedDrive.Path)

	fmt.Print("Enter current password: ")
	currentPassword, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Enter new password: ")
	newPassword1, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Confirm new password: ")
	newPassword2, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	if string(newPassword1) != string(newPassword2) {
		fmt.Println("New passwords do not match")
		os.Exit(1)
	}

	err = manager.ChangePassword(selectedDrive, currentPassword, newPassword1)
	if err != nil {
		fmt.Printf("Change password failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Password changed successfully")
}

func handleUnlockPro(args []string) {
	fmt.Println("unlock-pro not yet implemented")
	os.Exit(1)
}

func handleLockPro(args []string) {
	fmt.Println("lock-pro not yet implemented")
	os.Exit(1)
}

func handleChangePasswordPro(args []string) {
	fmt.Println("change-password-pro not yet implemented")
	os.Exit(1)
}

func handleStatus(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto status <drive-path>")
		os.Exit(1)
	}

	manager := drive.NewManager()
	drivePath := manager.ResolvePath(args[0])

	// Get drive info first
	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	var driveInfo *drive.Drive
	for _, d := range drives {
		if d.Path == drivePath {
			driveInfo = &d
			break
		}
	}

	// Check encryption status
	status, err := manager.CheckStatus(drivePath)
	if err != nil {
		fmt.Printf("Error checking status: %v\n", err)
		os.Exit(1)
	}

	// Print retro status box
	fmt.Print("\033[32m")
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║  ENCRYPTO STATUS CHECK                   ║")
	fmt.Println("║                                          ║")

	if driveInfo != nil {
		fmt.Printf("║  DEVICE: %-33s║\n", driveInfo.Name)
		fmt.Printf("║  PATH:   %-33s║\n", drivePath)
	} else {
		fmt.Printf("║  PATH:   %-33s║\n", drivePath)
	}

	fmt.Println("║                                          ║")

	if status.IsEncrypted {
		fmt.Println("║  STATUS: 🔒 ENCRYPTED                    ║")
		switch status.Method {
		case "apfs":
			fmt.Println("║  METHOD:  FileVault (APFS native)        ║")
		case "raw":
			fmt.Println("║  METHOD:  encrypt-pro (raw sector)       ║")
			fmt.Printf("║  VERSION: %-32d║\n", status.Version)
			if status.HasHidden {
				fmt.Println("║  HIDDEN:  Yes                            ║")
			} else {
				fmt.Println("║  HIDDEN:  No                             ║")
			}
		}
	} else {
		fmt.Println("║  STATUS: 🔓 NOT ENCRYPTED                ║")
	}

	fmt.Println("║                                          ║")

	if status.IsMounted {
		fmt.Println("║  MOUNTED: Yes                            ║")
	} else {
		fmt.Println("║  MOUNTED: No                             ║")
	}

	if status.Error != "" {
		fmt.Printf("║  ERROR: %-34s║\n", status.Error)
	}

	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Print("\033[0m")
}

func handleSize(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto size <path>")
		os.Exit(1)
	}

	target := strings.TrimSuffix(args[0], "/")

	// Raw device path — find in drive list
	if strings.HasPrefix(target, "/dev/") {
		manager := drive.NewManager()
		drives, err := manager.List()
		if err != nil {
			fmt.Printf("Error listing drives: %v\n", err)
			os.Exit(1)
		}
		for _, d := range drives {
			if d.Path == target {
				fmt.Printf("%s  %s\n", target, formatSize(d.Size))
				return
			}
		}
		fmt.Printf("Device not found: %s\n", target)
		os.Exit(1)
	}

	info, err := os.Stat(target)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Regular file
	if !info.IsDir() {
		fmt.Printf("%s  %s\n", filepath.Base(target), formatSize(uint64(info.Size())))
		return
	}

	// Directory or volume — show volume stats via statfs
	var stat syscall.Statfs_t
	if err := syscall.Statfs(target, &stat); err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		os.Exit(1)
	}

	blockSize := uint64(stat.Bsize)
	total := stat.Blocks * blockSize
	free := stat.Bavail * blockSize
	used := (stat.Blocks - stat.Bfree) * blockSize

	fmt.Printf("Path:  %s\n", target)
	fmt.Printf("Total: %s\n", formatSize(total))
	fmt.Printf("Used:  %s\n", formatSize(used))
	fmt.Printf("Free:  %s\n", formatSize(free))
}

func formatSize(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

type EncryptoCrypto struct{}

func (e *EncryptoCrypto) EncryptWithHidden(data, primaryPassword, hiddenPassword []byte) ([]byte, error) {
	_, err := crypto.DeriveKey(string(primaryPassword), []byte("encrypto"))
	if err != nil {
		return nil, err
	}
	_, err = crypto.DeriveKey(string(hiddenPassword), []byte("hidden"))
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(data))
	copy(ciphertext, data)

	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return []byte(encoded), nil
}

func handleEncryptHidden(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto encrypt-hidden <drive-path>")
		os.Exit(1)
	}

	manager := drive.NewManager()
	drivePath := manager.ResolvePath(args[0])

	drives, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing drives: %v\n", err)
		os.Exit(1)
	}

	var selectedDrive *drive.Drive
	for _, d := range drives {
		if d.Path == drivePath {
			selectedDrive = &d
			break
		}
	}

	if selectedDrive == nil {
		fmt.Printf("Drive not found: %s\n", drivePath)
		os.Exit(1)
	}

	if !selectedDrive.IsRemovable {
		fmt.Printf("Error: %s is not a removable drive\n", drivePath)
		os.Exit(1)
	}

	fmt.Printf("Encrypting %s with hidden volume...\n", selectedDrive.Name)
	fmt.Println("Warning: This will encrypt all data on the drive.")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you sure? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "yes" {
		fmt.Println("Encryption cancelled")
		return
	}

	fmt.Print("Enter main password: ")
	password1, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Confirm main password: ")
	password2, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	if string(password1) != string(password2) {
		fmt.Println("Passwords do not match")
		os.Exit(1)
	}

	fmt.Print("Enter hidden volume password: ")
	hiddenPassword1, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Confirm hidden password: ")
	hiddenPassword2, err := readPassword()
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	if string(hiddenPassword1) != string(hiddenPassword2) {
		fmt.Println("Hidden passwords do not match")
		os.Exit(1)
	}

	err = manager.EncryptWithHidden(selectedDrive, password1, hiddenPassword1)
	if err != nil {
		fmt.Printf("Encryption failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Drive encrypted with hidden volume successfully")
}

func readPassword() ([]byte, error) {
	// Try term.ReadPassword first (hides input) - only works in terminal
	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		password, err := term.ReadPassword(fd)
		if err == nil {
			fmt.Println()
			return password, nil
		}
	}
	// Fallback: read from stdin using fmt.Scan (works with pipes)
	var password string
	_, err := fmt.Scan(&password)
	if err != nil {
		return nil, err
	}
	return []byte(password), nil
}

var encrypto = &EncryptoCrypto{}
