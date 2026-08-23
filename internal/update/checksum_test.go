package update

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestParseAndVerifyChecksums(t *testing.T) {
	t.Parallel()

	data := []byte("hello")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])
	text := hexSum + "  msc_1.0.0_linux_amd64.tar.gz\n" +
		strings.Repeat("0", 64) + "  other.tar.gz\n"

	got := ParseChecksums(text)
	if got["msc_1.0.0_linux_amd64.tar.gz"] != hexSum {
		t.Fatalf("%v", got)
	}
	if err := VerifySHA256(data, hexSum); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(data, strings.Repeat("ab", 32)); err == nil {
		t.Fatal("expected mismatch")
	}
}
