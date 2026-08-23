package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ParseChecksums reads a GoReleaser checksums.txt (sha256  filename).
func ParseChecksums(text string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		out[fields[len(fields)-1]] = strings.ToLower(fields[0])
	}
	return out
}

// VerifySHA256 compares data to a hex digest.
func VerifySHA256(data []byte, wantHex string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, strings.TrimSpace(wantHex)) {
		return fmt.Errorf("checksum mismatch: got %s, want %s", got, wantHex)
	}
	return nil
}
