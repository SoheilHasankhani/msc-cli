package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// Fetcher downloads a URL's body. Tests inject a fake; production uses HTTPFetch.
type Fetcher func(ctx context.Context, url string) ([]byte, error)

// HTTPFetch returns a Fetcher that GETs URLs with the given client.
func HTTPFetch(client *http.Client) Fetcher {
	if client == nil {
		client = http.DefaultClient
	}
	return func(ctx context.Context, rawURL string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "msc-cli")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("download %s: HTTP %d", rawURL, resp.StatusCode)
		}
		return io.ReadAll(io.LimitReader(resp.Body, maxArchiveBytes))
	}
}

// Apply downloads the planned archive, verifies its checksum, and replaces dest.
func Apply(ctx context.Context, p Plan, dest, goos string, fetch Fetcher) error {
	if fetch == nil {
		return fmt.Errorf("no downloader")
	}
	if dest == "" {
		return fmt.Errorf("install path is empty")
	}
	if p.AssetURL == "" || p.ChecksumsURL == "" || p.AssetName == "" {
		return fmt.Errorf("update plan is missing asset URLs")
	}
	sums, err := fetch(ctx, p.ChecksumsURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	archive, err := fetch(ctx, p.AssetURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", p.AssetName, err)
	}
	table := ParseChecksums(string(sums))
	want, ok := table[p.AssetName]
	if !ok {
		return fmt.Errorf("checksums.txt has no entry for %s", p.AssetName)
	}
	if err := VerifySHA256(archive, want); err != nil {
		return err
	}
	bin, err := ExtractBinary(archive, goos)
	if err != nil {
		return err
	}
	if len(bin) == 0 {
		return fmt.Errorf("extracted binary is empty")
	}
	return ReplaceExecutable(dest, bin)
}
