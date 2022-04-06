package lepton

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

// Apply performs an update of the current executable (or opts.TargetFile, if set) with the contents of the given io.Reader.
//
// Apply performs the following actions to ensure a safe cross-platform update:
//
// 1. If configured, applies the contents of the update io.Reader as a binary patch.
//
// 2. If configured, computes the checksum of the new executable and verifies it matches.
//
// 3. If configured, verifies the signature with a public key.
//
// 4. Creates a new file, /path/to/.target.new with the TargetMode with the contents of the updated file
//
// 5. Renames /path/to/target to /path/to/.target.old
//
// 6. Renames /path/to/.target.new to /path/to/target
//
// 7. If the final rename is successful, deletes /path/to/.target.old, returns no error. On Windows,
// the removal of /path/to/target.old always fails, so instead Apply hides the old file instead.
//
// 8. If the final rename fails, attempts to roll back by renaming /path/to/.target.old
// back to /path/to/target.
func Apply(update io.Reader, opts Options) error {

	// set defaults
	if opts.Hash == 0 {
		opts.Hash = crypto.SHA256
	}

	if opts.TargetMode == 0 {
		opts.TargetMode = 0755
	}

	// get target path
	var err error
	opts.TargetPath, err = opts.getPath()
	if err != nil {
		return err
	}

	var newBytes []byte

	if newBytes, err = ioutil.ReadAll(update); err != nil {
		return err
	}

	// verify checksum if requested
	if opts.Checksum != nil {
		if err = opts.verifyChecksum(newBytes); err != nil {
			return err
		}
	}

	// get the directory the executable exists in
	updateDir := filepath.Dir(opts.TargetPath)
	filename := filepath.Base(opts.TargetPath)

	// Copy the contents of newbinary to a new executable file
	newPath := filepath.Join(updateDir, fmt.Sprintf(".%s.new", filename))
	fp, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, opts.TargetMode)
	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = io.Copy(fp, bytes.NewReader(newBytes))
	if err != nil {
		return err
	}

	// if we don't call fp.Close(), windows won't let us move the new executable
	// because the file will still be "in use"
	fp.Close()

	// this is where we'll move the executable to so that we can swap in the updated replacement
	oldPath := opts.OldSavePath
	removeOld := opts.OldSavePath == ""
	if removeOld {
		oldPath = filepath.Join(updateDir, fmt.Sprintf(".%s.old", filename))
	}

	_ = os.Remove(oldPath)

	// move the existing executable to a new file in the same directory
	err = os.Rename(opts.TargetPath, oldPath)
	if err != nil {
		return err
	}

	// move the new executable in to become the new program
	err = os.Rename(newPath, opts.TargetPath)

	if err != nil {
		// move unsuccessful
		//
		// The filesystem is now in a bad state. We have successfully
		// moved the existing binary to a new location, but we couldn't move the new
		// binary to take its place. That means there is no file where the current executable binary
		// used to be!
		// Try to rollback by restoring the old binary to its original path.
		rerr := os.Rename(oldPath, opts.TargetPath)
		if rerr != nil {
			return &rollbackErr{err, rerr}
		}

		return err
	}

	// move successful, remove the old binary if needed
	if removeOld {
		os.Remove(oldPath)
	}

	return nil
}

// RollbackError takes an error value returned by Apply and returns the error, if any,
// that occurred when attempting to roll back from a failed update. Applications should
// always call this function on any non-nil errors returned by Apply.
//
// If no rollback was needed or if the rollback was successful, RollbackError returns nil,
// otherwise it returns the error encountered when trying to roll back.
func RollbackError(err error) error {
	if err == nil {
		return nil
	}
	if rerr, ok := err.(*rollbackErr); ok {
		return rerr.rollbackErr
	}
	return nil
}

type rollbackErr struct {
	error             // original error
	rollbackErr error // error encountered while rolling back
}

// Options for binary update
type Options struct {
	// TargetPath defines the path to the file to update.
	// The empty string means 'the executable file of the running program'.
	TargetPath string

	// Create TargetPath replacement with this file mode. If zero, defaults to 0755.
	TargetMode os.FileMode

	// Checksum of the new binary to verify against. If nil, no checksum or signature verification is done.
	Checksum []byte

	// Public key to use for signature verification. If nil, no signature verification is done.
	PublicKey crypto.PublicKey

	// Signature to verify the updated file. If nil, no signature verification is done.
	Signature []byte

	// Use this hash function to generate the checksum. If not set, SHA256 is used.
	Hash crypto.Hash

	// Store the old executable file at this path after a successful update.
	// The empty string means the old executable file will be removed after the update.
	OldSavePath string
}

// CheckPermissions determines whether the process has the correct permissions to
// perform the requested update. If the update can proceed, it returns nil, otherwise
// it returns the error that would occur if an update were attempted.
func (o *Options) CheckPermissions() error {
	// get the directory the file exists in
	path, err := o.getPath()
	if err != nil {
		return err
	}

	fileDir := filepath.Dir(path)
	fileName := filepath.Base(path)

	// attempt to open a file in the file's directory
	newPath := filepath.Join(fileDir, fmt.Sprintf(".%s.new", fileName))
	fp, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, o.TargetMode)
	if err != nil {
		return err
	}
	fp.Close()

	_ = os.Remove(newPath)
	return nil
}

// SetPublicKeyPEM is a convenience method to set the PublicKey property
// used for checking a completed update's signature by parsing a
// Public Key formatted as PEM data.
func (o *Options) SetPublicKeyPEM(pembytes []byte) error {
	block, _ := pem.Decode(pembytes)
	if block == nil {
		return errors.New("couldn't parse PEM data")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}
	o.PublicKey = pub
	return nil
}

func (o *Options) getPath() (string, error) {
	if o.TargetPath == "" {
		return os.Executable()
	}
	return o.TargetPath, nil
}

func (o *Options) verifyChecksum(updated []byte) error {
	checksum, err := checksumFor(o.Hash, updated)
	if err != nil {
		return err
	}

	if !bytes.Equal(o.Checksum, checksum) {
		return fmt.Errorf("Updated file has wrong checksum. Expected: %x, got: %x", o.Checksum, checksum)
	}
	return nil
}

func checksumFor(h crypto.Hash, payload []byte) ([]byte, error) {
	if !h.Available() {
		return nil, errors.New("requested hash function not available")
	}
	hash := h.New()
	hash.Write(payload) // guaranteed not to error
	return hash.Sum([]byte{}), nil
}

// DoUpdate updates file using provided URL
func DoUpdate(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return Apply(resp.Body, Options{})
}
