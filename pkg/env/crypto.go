package env

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
)

// GenerateRandomPassword generates a cryptographically secure random password
// of the specified length using alphanumeric characters.
func GenerateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

// VerifySHA256 compares the SHA256 checksum of the file at filePath with the expectedHash.
func VerifySHA256(filePath, expectedHash string) (bool, error) {
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return false, fmt.Errorf("failed to open file for hash verification: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	actualHash := hex.EncodeToString(hash.Sum(nil))
	return actualHash == expectedHash, nil
}

// DownloadFile downloads a file from the specified URL to the local filePath.
func DownloadFile(url, filePath string) error {
	resp, err := http.Get(url) // #nosec G107
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: status code %d", resp.StatusCode)
	}

	out, err := os.Create(filePath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to save downloaded file: %w", err)
	}

	return nil
}
