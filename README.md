# 🔐 encrypto

Cross-platform full disk encryption for external drives (similar to BitLocker/FileVault).

## ✨ Features

- **Full Disk Encryption (FDE)** using AES-256-XTS for maximum security
- **File Encryption** for instant encryption of individual files
- **Password protection** with Argon2id key derivation (military-grade)
- **Cross-platform** support (macOS, Windows, Linux)
- **Hidden volume support** for plausible deniability
- **Retro 80s-style progress bar** with real-time updates
- **Safe interruption handling** - Ctrl+C safely aborts without corrupting data

## 🚀 Quick Start

### Building

```bash
go build -o encrypto ./cmd/encrypto
```

### Basic Commands

```bash
# See all available drives
./encrypto list

# Check if a drive is encrypted
./encrypto status /dev/disk4
```

## 📁 File Encryption (Instant)

Encrypt individual files in **seconds** - perfect for testing and daily use:

```bash
# Encrypt a file
./encrypto encrypt-file secret.txt secret.txt.enc
# Enter password: ********
# Confirm password: ********
# ✓ File encrypted: secret.txt.enc

# Decrypt it back
./encrypto decrypt-file secret.txt.enc secret_restored.txt
# Enter password: ********
# ✓ File decrypted: secret_restored.txt
```

**How it works:**
- Uses AES-256-GCM encryption
- Adds 40-byte header + authentication tag
- Argon2id password hashing
- Works on any file type (documents, images, videos, etc.)

## 💾 Full Disk Encryption

**⚠️ Warning:** This will encrypt your ENTIRE drive and takes 30-60 minutes for large drives.

### Encrypt a Drive (One-Time Setup)

```bash
# 1. Unmount the drive first (macOS example)
diskutil unmountDisk /dev/disk4

# 2. Encrypt the entire drive
sudo ./encrypto encrypt /dev/disk4

# You will see:
# Encrypting SanDisk 3.2Gen1 (/dev/disk4)...
# Warning: This will encrypt all data on the drive.
# Are you sure? (yes/no): yes
# Enter password: ********
# Confirm password: ********
# 
# [Green retro progress bar appears]
# ╔══════════════════════════════════════════╗
# ║  ENCRYPTO v1.0 - ENCRYPTING              ║
# ║  [████████████████████░░░░░░░░░░░] 45.2% ║
# ║  BYTES:   105.3 / 232.0 GB               ║
# ║  SPEED:   78.5 MB/s                      ║
# ║  ETA:     00:14:32                       ║
# ╚══════════════════════════════════════════╝
#
# Drive encrypted successfully
```

### Daily Use (After Encryption)

**Lock/Unlock Workflow:**

```bash
# Unlock the drive (mount it)
sudo ./encrypto unlock /dev/disk4
# Enter password: ********
# Drive unlocked successfully

# The drive is now mounted and ready to use
# Use your files normally through Finder/Explorer

# When finished, lock the drive (unmount it)
sudo ./encrypto lock /dev/disk4
# Drive locked successfully
```

**Key Points:**
- **Encrypt/Decrypt**: One-time setup, takes 30-60 minutes
- **Lock/Unlock**: Daily use, takes 2-5 seconds
- After unlocking, use the drive like normal
- Files remain encrypted on disk, decrypted on-the-fly

### Decrypt a Drive (Full Decryption)

```bash
# Decrypt the entire drive back to plaintext
sudo ./encrypto decrypt /dev/disk4
# Enter password: ********
# 
# [Green retro progress bar shows decryption progress]
#
# Drive decrypted successfully
```

## 🔍 Checking Status

```bash
./encrypto status /dev/disk4

# Output:
# ╔══════════════════════════════════════════╗
# ║  ENCRYPTO STATUS CHECK                   ║
# ║                                          ║
# ║  DEVICE: SanDisk 3.2Gen1                  ║
# ║  PATH:   /dev/disk4                       ║
# ║                                          ║
# ║  STATUS: 🔒 ENCRYPTED                    ║
# ║  VERSION: 1                               ║
# ║  HIDDEN:   No                             ║
# ║                                          ║
# ║  MOUNTED: No                             ║
# ╚══════════════════════════════════════════╝
```

## 📋 Command Reference

### Drive Commands (Full Disk Encryption)

| Command | Description | Time | Usage |
|---------|-------------|------|-------|
| `encrypt` | Encrypt entire drive | 30-60 min | `sudo ./encrypto encrypt /dev/diskX` |
| `decrypt` | Decrypt entire drive | 30-60 min | `sudo ./encrypto decrypt /dev/diskX` |
| `unlock` | Mount encrypted drive | 2-5 sec | `sudo ./encrypto unlock /dev/diskX` |
| `lock` | Unmount drive | 2-5 sec | `sudo ./encrypto lock /dev/diskX` |
| `status` | Check encryption status | Instant | `./encrypto status /dev/diskX` |

### File Commands (Instant Encryption)

| Command | Description | Time | Usage |
|---------|-------------|------|-------|
| `encrypt-file` | Encrypt a single file | Instant | `./encrypto encrypt-file input.txt output.enc` |
| `decrypt-file` | Decrypt a single file | Instant | `./encrypto decrypt-file input.enc output.txt` |

### Utility Commands

| Command | Description |
|---------|-------------|
| `list` | Show all available drives |
| `encrypt-hidden` | Encrypt with hidden volume (plausible deniability) |

## 🖥️ Platform-Specific Notes

### macOS
- Drives appear as `/dev/disk0`, `/dev/disk1`, etc.
- Use `diskutil list` to see drives
- Requires `sudo` for raw disk access
- Unmount first: `diskutil unmountDisk /dev/diskX`

### Linux
- Drives appear as `/dev/sda`, `/dev/sdb`, etc.
- Use `lsblk` to see drives
- Requires `sudo` for raw disk access
- Unmount first: `umount /dev/sdX1`

### Windows
- Drives appear as `\\.\PhysicalDrive0`, `D:`, etc.
- Run as Administrator
- Drives shown as `C:`, `D:`, etc.

## ⚠️ Important Warnings

1. **Backup First**: Always backup important data before encrypting
2. **Don't Forget Password**: If you lose the password, data is **PERMANENTLY** lost
3. **Interrupt Safety**: Press Ctrl+C anytime - encryption safely aborts
4. **One-Time Encryption**: After encrypting, use `lock`/`unlock` for daily use (not re-encrypt)
5. **sudo Required**: All drive operations require administrator privileges

## 🏗️ Architecture

- **Crypto**: AES-256-XTS (drives) / AES-256-GCM (files) with Argon2id key derivation
- **Sector Size**: 512 bytes (standard for disk encryption)
- **Header**: 512-byte metadata block at start of drive
- **Sync**: Every sector written is immediately synced to disk (crash-safe)

## 📝 License

MIT License - See LICENSE file for details

## 🐛 Troubleshooting

**"Permission denied" error:**
- Use `sudo` before the command
- Make sure drive is unmounted first

**"Resource busy" error:**
- Unmount the drive: `diskutil unmountDisk /dev/diskX` (macOS)

**"Already encrypted" error:**
- Drive already has encrypto header
- Use `unlock` instead of `encrypt`

**Progress bar freezes:**
- Normal for large drives (30-60 minutes)
- Drive LED should be blinking
- Press Ctrl+C to cancel safely

## 💡 Tips

- **Test first**: Use `encrypt-file` on a test file before encrypting your drive
- **Strong passwords**: Use 12+ characters with mixed case, numbers, symbols
- **Write down password**: Store in a safe place (password manager)
- **Eject properly**: Always use `lock` command before physically removing drive
