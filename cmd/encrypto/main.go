package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/encrypto/encrypto/internal/crypto"
	"github.com/encrypto/encrypto/internal/drive"
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
	case "unlock":
		handleUnlock(os.Args[2:])
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
	fmt.Println("  encrypto encrypt <drive-path>     Encrypt a drive")
	fmt.Println("  encrypto decrypt <drive-path>     Decrypt a drive")
	fmt.Println("  encrypto unlock <drive-path>      Unlock an encrypted drive")
	fmt.Println("  encrypto list                     List available drives")
	fmt.Println("  encrypto encrypt-hidden <drive>   Encrypt with hidden volume")
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

	fmt.Printf("Encrypting %s with hidden volume...\n", selectedDrive.Name, selectedDrive.Path)
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

	fmt.Print("Enter hidden volume password (can be different): ")
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
	fmt.Scanln()
	password := make([]byte, 0)
	for {
		b := make([]byte, 1)
		n, err := os.Stdin.Read(b)
		if n > 0 {
			switch b[0] {
			case '\n', '\r':
				return password, nil
			case 8, 127:
				if len(password) > 0 {
					password = password[:len(password)-1]
				}
			default:
				password = append(password, b[0])
			}
		}
		if err != nil {
			return nil, err
		}
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

var encrypto = &EncryptoCrypto{}
