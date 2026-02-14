# encrypto

Cross-platform full disk encryption for external drives (similar to BitLocker).

**Features:**
- Full Disk Encryption (FDE) using AES-256-XTS
- Password protection with PBKDF2/Argon2id key derivation  
- Cross-platform compatibility (Windows, macOS, Linux)
- Hidden volume support for plausible deniability
- Native password prompts on unlock

## Usage

```bash
# List available drives
encrypto list

# Encrypt an external drive
encrypto encrypt /dev/sdX

# Unlock encrypted drive
encrypto unlock /dev/sdX
```

## Architecture

- **Crypto**: AES-256-XTS encryption with Argon2id key derivation
- **Drive Access**: Platform-specific drive enumeration and operations
- **Header Format**: Custom metadata format for cross-platform compatibility
- **Password UI**: Native prompts on each platform

## Building

```bash
go build -o encrypto ./cmd/encrypto
```

## License

MIT
