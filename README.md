# encrypto

Password-protect external drives and encrypt individual files. Drive encryption
uses macOS native FileVault (APFS) so your existing data is preserved and the
drive works normally after unlocking. File encryption uses AES-256-GCM with
Argon2id key derivation — standard algorithms you can decrypt independently of
this tool if needed.


## Paths

All commands accept any of these path formats and resolve them automatically:

    /dev/disk4
    /Volumes/Extreme SSD
    /Volumes/Extreme SSD/

Use `./encrypto list` to see all connected drives with their device paths and
mount points.


## Drive encryption

Encryption is a one-time setup. After it is done you unlock and lock the drive
for daily use — you do not re-encrypt it each time.

The drive must be formatted as APFS. Most modern external drives on macOS
already are. If yours is not, reformat it in Disk Utility (Erase, choose APFS)
before proceeding. Reformatting erases the drive, so back up first.

Enable encryption:

    sudo ./encrypto encrypt "/Volumes/Extreme SSD"

Your existing files are not deleted. Encryption runs in the background and the
drive is immediately usable. On a large drive the background conversion may take
some time, but you do not need to wait for it.

Check status:

    ./encrypto status "/Volumes/Extreme SSD"


## Daily use

Unlock the drive when you plug it in:

    sudo ./encrypto unlock "/Volumes/Extreme SSD"

Enter your password. The drive mounts and appears in Finder as normal.

Lock it when you are done:

    sudo ./encrypto lock "/Volumes/Extreme SSD"

The drive unmounts and is inaccessible without the password. It is safe to
physically remove it after locking.


## Remove encryption

To go back to a plain unencrypted drive:

    sudo ./encrypto decrypt "/Volumes/Extreme SSD"

Your data is preserved throughout. The drive runs in the background converting
back to unencrypted, same as the initial encryption.


## File encryption

Encrypting a single file is instant. Good for testing your password before
committing to a full drive operation.

Encrypt a file (creates a new encrypted copy):

    ./encrypto encrypt-file photo.jpg photo.jpg.enc

Encrypt in-place (replaces the original):

    ./encrypto encrypt-file photo.jpg

Decrypt:

    ./encrypto decrypt-file photo.jpg.enc photo_restored.jpg

Test that the restored file is identical:

    diff photo.jpg photo_restored.jpg

No output means byte-for-byte identical.


## Check drive or folder size

    ./encrypto size "/Volumes/Extreme SSD"
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

Drive encryption uses macOS FileVault (AES-XTS-128) and can be managed directly
through Disk Utility or `diskutil` as with any FileVault volume.


## Quick reference

    ./encrypto list                                    list all drives
    ./encrypto status  "/Volumes/Name"                 check encryption status
    ./encrypto size    "/Volumes/Name"                 show capacity and usage

    sudo ./encrypto encrypt  "/Volumes/Name"           enable FileVault (one-time)
    sudo ./encrypto decrypt  "/Volumes/Name"           remove FileVault
    sudo ./encrypto unlock   "/Volumes/Name"           mount encrypted drive
    sudo ./encrypto lock     "/Volumes/Name"           unmount encrypted drive

    ./encrypto encrypt-file  input.txt  input.txt.enc  encrypt a file
    ./encrypto decrypt-file  input.txt.enc  output.txt  decrypt a file


## Troubleshooting

"Permission denied" — add sudo before the command.

"Drive not APFS formatted" — open Disk Utility, select the drive, click Erase,
choose APFS as the format. This erases the drive, so back up first. After
formatting, run encrypt again.

"Wrong password" — the password does not match. There is no way to recover
access without the correct password.


## Advanced: raw sector encryption

`encrypt-pro` and `decrypt-pro` perform direct sector-by-sector encryption of
the raw block device. This overwrites the partition table and renders the drive
completely inaccessible to any OS until `decrypt-pro` is run. It is intended for
research and forensic use cases, not general use.

    sudo ./encrypto encrypt-pro  /dev/disk4   raw sector encrypt (DESTRUCTIVE)
    sudo ./encrypto decrypt-pro  /dev/disk4   raw sector decrypt
    sudo ./encrypto encrypt-hidden /dev/disk4  raw sector encrypt with hidden volume

Both commands require `YES` confirmation and display an explicit warning before
proceeding. Back up all data first.
