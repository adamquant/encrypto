# encrypto

Password-protect external drives and encrypt individual files. encrypto provides two
different approaches to drive encryption depending on your needs.


## Which encryption mode should I use?

### Mode 1: FileVault (encrypt / decrypt)

**Use this if:**
- You only use Macs
- You want the simplest option
- You want encryption that "just works" with macOS

**How it works:**
- Uses macOS native FileVault (APFS encryption)
- Your existing data is preserved
- Encryption runs in the background (can take hours on large drives)
- After setup, just `unlock` to mount, `lock` to unmount

**Limitations:**
- Only works on Mac
- Background conversion can get stuck (drive still works)
- Cannot access on Windows or Linux

### Mode 2: encrypt-pro (encrypt-pro / decrypt-pro)

**Use this if:**
- You want cross-platform compatibility (Mac, Windows, Linux)
- You want true portable security
- You're comfortable with CLI tools

**How it works:**
- Encrypts every sector on the drive individually
- The OS cannot read the drive at all when locked
- Only this tool can unlock it with your password
- Data is NOT deleted — just encrypted beyond OS access

**Advantages:**
- Works on ANY computer (Mac, Windows, Linux)
- No background conversion — instant encryption
- No OS-level dependencies

---

## Quick Decision Guide

| Question | Answer | Use |
|----------|--------|-----|
| Will you only use Mac? | Yes | `encrypt` (FileVault) |
| Need to use on Windows/Linux? | Yes | `encrypt-pro` |
| Want simplest option? | Yes | `encrypt` |
| Want maximum security/portability? | Yes | `encrypt-pro` |


## Paths

All commands accept any of these path formats and resolve them automatically:

    /dev/disk4
    /Volumes/MyDrive
    /Volumes/MyDrive/

Use `./encrypto list` to see all connected drives with their device paths and
mount points.


## Mode 1: FileVault Encryption (Mac Only)

The drive must be formatted as APFS. Most modern external drives on macOS
already are. If yours is not, reformat it in Disk Utility (Erase, choose APFS)
before proceeding. Reformatting erases the drive, so back up first.

### Enable encryption:

    sudo ./encrypto encrypt "/Volumes/MyDrive"

Your existing files are not deleted. Encryption runs in the background and the
drive is immediately usable. On a large drive the background conversion may take
hours or days, but you do not need to wait for it.

### Daily use (lock/unlock):

    # When you plug in the drive, unlock it:
    sudo ./encrypto unlock "/Volumes/MyDrive"
    # Enter your password. The drive mounts in Finder.

    # When you're done, lock it:
    sudo ./encrypto lock "/Volumes/MyDrive"

The drive unmounts and is inaccessible without the password. It is safe to
physically remove it after locking.

### Check status:

    ./encrypto status "/Volumes/MyDrive"

### Remove encryption entirely:

    sudo ./encrypto decrypt "/Volumes/MyDrive"

Your data is preserved throughout. The drive runs in the background converting
back to unencrypted, same as the initial encryption.

### Change password:

    sudo ./encrypto change-password "/Volumes/MyDrive"

Enter your current password, then enter a new password twice to confirm.


## Mode 2: Raw Sector Encryption (Cross-Platform)

This encrypts the drive at the sector level. When locked, NO operating system
can read it — not Mac, not Windows, not Linux. Only encrypto with your password
can unlock it.

### Enable encryption:

    sudo ./encrypto encrypt-pro /dev/disk4

**Warning:** This command will make the drive invisible to your operating system.
You MUST use `./encrypto decrypt-pro` to restore access. Back up all data first.

The command will ask you to type `YES` in capitals to confirm.

### Daily use (lock/unlock):

    # Unlock to mount:
    sudo ./encrypto unlock-pro /dev/disk4
    # Enter your password. Drive mounts in Finder.

    # Lock to secure:
    sudo ./encrypto lock-pro /dev/disk4

### Remove encryption entirely:

    sudo ./encrypto decrypt-pro /dev/disk4

This decrypts all sectors and restores the drive to normal. Your data is preserved.

### Change password:

    sudo ./encrypto change-password-pro /dev/disk4

This re-encrypts the entire drive with your new password. It takes the same
time as initial encryption (encrypts every sector).


## File Encryption

Encrypting a single file is instant. Good for testing your password before
committing to a full drive operation, or for encrypting individual sensitive files.

### Encrypt a file (creates a new encrypted copy):

    ./encrypto encrypt-file photo.jpg photo.jpg.enc

### Encrypt in-place (replaces the original):

    ./encrypto encrypt-file photo.jpg

### Decrypt:

    ./encrypto decrypt-file photo.jpg.enc photo_restored.jpg

### Test that the restored file is identical:

    diff photo.jpg photo_restored.jpg

No output means byte-for-byte identical.


## Check drive or folder size

    ./encrypto size "/Volumes/MyDrive"
    ./encrypto size /dev/disk4
    ./encrypto size ~/Documents
    ./encrypto size report.pdf

For volumes and directories, shows total capacity, used, and free. For files,
shows file size.


## Passwords

- Use a long passphrase you can remember, not a random string you will forget
- Write it down and store it somewhere physically safe
- There is no password reset or recovery. A forgotten password means permanent
  loss of access to the drive
- Test encrypt and decrypt a single file before encrypting your drive to confirm
  you have the password memorised correctly


## Decrypting files without this repo

File encryption uses published standards available in any language:

    Encryption:     AES-256-GCM  (NIST SP 800-38D)
    Key derivation: Argon2id     (RFC 9106)
    Parameters:     time=3, memory=64MB, parallelism=4, salt=16 bytes

The salt and parameters are stored in the first bytes of every encrypted file.
Python libraries that implement these:

    pip install cryptography argon2-cffi

As long as you remember your password, you can decrypt any file independently
of this tool.

Drive encryption modes:
- FileVault uses macOS native encryption (AES-XTS-128) — can be managed via Disk Utility
- encrypt-pro uses AES-256 sector encryption — only this tool can decrypt


## Quick reference

    # List drives
    ./encrypto list

    # Check encryption status
    ./encrypto status "/Volumes/Name"
    ./encrypto status /dev/disk4

    # Show capacity and usage
    ./encrypto size "/Volumes/Name"

    # --- FileVault Mode (Mac only) ---
    sudo ./encrypto encrypt "/Volumes/Name"         enable FileVault
    sudo ./encrypto decrypt "/Volumes/Name"         remove FileVault
    sudo ./encrypto unlock "/Volumes/Name"          mount encrypted drive
    sudo ./encrypto lock "/Volumes/Name"            unmount encrypted drive
    sudo ./encrypto change-password "/Volumes/Name"  change FileVault password

    # --- encrypt-pro Mode (Cross-platform) ---
    sudo ./encrypto encrypt-pro /dev/disk4          raw sector encrypt (DESTRUCTIVE)
    sudo ./encrypto decrypt-pro /dev/disk4          raw sector decrypt
    sudo ./encrypto unlock-pro /dev/disk4           mount encrypted drive
    sudo ./encrypto lock-pro /dev/disk4            unmount encrypted drive
    sudo ./encrypto change-password-pro /dev/disk4   change encrypt-pro password

    # --- File encryption ---
    ./encrypto encrypt-file input.txt output.txt.enc
    ./encrypto decrypt-file input.txt.enc output.txt


## Troubleshooting

"Permission denied" — add sudo before the command.

"Drive not APFS formatted" — open Disk Utility, select the drive, click Erase,
choose APFS as the format. This erases the drive, so back up first. After
formatting, run encrypt again.

"Wrong password" — the password does not match. There is no way to recover
access without the correct password.

To check FileVault encryption progress manually:

    diskutil apfs list

Look for your volume under the APFS Container. The "Encryption Progress" field
shows conversion progress (e.g., "10.0% (Paused)" means it's 10% complete).

If encrypt-pro says "operation not supported" — the drive may already be encrypted
or is not a valid target. Use `./encrypto status /dev/diskX` to check.


## Advanced: hidden volume

`encrypt-hidden` creates a drive with two passwords — a "decoy" volume visible
to anyone, and a hidden volume that only appears with the second password. This
provides plausible deniability.

    sudo ./encrypto encrypt-hidden /dev/disk4

Warning: Use with caution. If you forget both passwords, there is no recovery.
