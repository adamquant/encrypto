package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
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
	case "encrypt-file":
		handleEncryptFile(os.Args[2:])
	case "decrypt-file":
		handleDecryptFile(os.Args[2:])
	case "unlock":
		handleUnlock(os.Args[2:])
	case "lock":
		handleLock(os.Args[2:])
	case "status":
		handleStatus(os.Args[2:])
	case "list":
		handleList()
	case "encrypt-hidden":
		handleEncryptHidden(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("encrypto - Cross-platform full disk encryption")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  encrypto encrypt <drive-path>          Encrypt a drive (60+ min)")
	fmt.Println("  encrypto decrypt <drive-path>          Decrypt a drive (60+ min)")
	fmt.Println("  encrypt-file <file> <output>           Encrypt a file (instant)")
	fmt.Println("  decrypt-file <file> <output>           Decrypt a file (instant)")
	fmt.Println("  encrypto unlock <drive-path>           Unlock an encrypted drive")
	fmt.Println("  encrypto lock <drive-path>             Lock (unmount) a drive")
	fmt.Println("  encrypto status <drive-path>           Check drive encryption status")
	fmt.Println("  encrypto list                          List available drives")
	fmt.Println("  encrypto encrypt-hidden <drive>        Encrypt with hidden volume")
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

	drivePath := args[0]

	manager := drive.NewManager()
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

	fmt.Printf("Encrypting %s (%s)...\n", selectedDrive.Name, selectedDrive.Path)
	fmt.Println("Warning: This will encrypt all data on the drive.")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you sure? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "yes" {
		fmt.Println("Encryption cancelled")
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

	fmt.Println("Drive encrypted successfully")
}

func handleDecrypt(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto decrypt <drive-path>")
		os.Exit(1)
	}

	drivePath := args[0]

	manager := drive.NewManager()
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

	fmt.Printf("Decrypting %s (%s)...\n", selectedDrive.Name, selectedDrive.Path)
	fmt.Println("Warning: This will decrypt all data on the drive.")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you sure? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "yes" {
		fmt.Println("Decryption cancelled")
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

	fmt.Println("Drive decrypted successfully")
}

func handleEncryptFile(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: encrypto encrypt-file <input-file> <output-file>")
		os.Exit(1)
	}

	inputPath := args[0]
	outputPath := args[1]

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

	fmt.Printf("File encrypted successfully: %s\n", outputPath)
}

func handleDecryptFile(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: encrypto decrypt-file <input-file> <output-file>")
		os.Exit(1)
	}

	inputPath := args[0]
	outputPath := args[1]

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

	fmt.Printf("File decrypted successfully: %s\n", outputPath)
}

func handleLock(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto lock <drive-path>")
		os.Exit(1)
	}

	drivePath := args[0]

	manager := drive.NewManager()
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

	drivePath := args[0]

	manager := drive.NewManager()
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

func handleStatus(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: encrypto status <drive-path>")
		os.Exit(1)
	}

	drivePath := args[0]

	manager := drive.NewManager()

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
		fmt.Printf("║  VERSION: %-32d║\n", status.Version)
		if status.HasHidden {
			fmt.Println("║  HIDDEN:   Yes                           ║")
		} else {
			fmt.Println("║  HIDDEN:   No                            ║")
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

	drivePath := args[0]

	manager := drive.NewManager()
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
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Newline after password input
	return password, err
}

var encrypto = &EncryptoCrypto{}
