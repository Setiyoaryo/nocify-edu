// Package media stores binary assets and runs the brand-asset governance
// lifecycle (§25, §44): every file has provenance and a license before it can
// be published, AI-generated imagery can never be published, and each kind
// has a performance budget.
package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Storage abstracts object storage. LocalStorage backs development and
// single-node deployments; an S3-compatible implementation satisfies the same
// interface when files move to object storage.
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader) (int64, error)
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
}

type LocalStorage struct{ Dir string }

func (s LocalStorage) path(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	if strings.Contains(clean, "..") {
		return "", errors.New("invalid key")
	}
	return filepath.Join(s.Dir, clean), nil
}

func (s LocalStorage) Put(_ context.Context, key string, r io.Reader) (int64, error) {
	p, err := s.path(key)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return 0, err
	}
	tmp := p + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return 0, err
	}
	return n, os.Rename(tmp, p)
}

func (s LocalStorage) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// AllowedMIME maps accepted upload types to file extensions.
var AllowedMIME = map[string]string{
	"image/svg+xml":    ".svg",
	"image/png":        ".png",
	"image/webp":       ".webp",
	"image/avif":       ".avif",
	"image/jpeg":       ".jpg",
	"application/json": ".json", // Lottie
	"audio/mpeg":       ".mp3",
	"audio/ogg":        ".ogg",
	"audio/webm":       ".webm",
}

// Budget returns the byte budget for an asset kind (§36 working budgets).
func Budget(kind string) int64 {
	switch kind {
	case "icon", "logo", "shape":
		return 50 << 10
	case "sticker", "reward":
		return 100 << 10
	case "animation", "sound":
		return 500 << 10
	}
	return 300 << 10 // illustrations, characters, backgrounds, diagrams, photos
}

// Lifecycle: imported → license_verified → design_approved → performance_checked → published.
var nextStatus = map[string]string{
	"imported":            "license_verified",
	"license_verified":    "design_approved",
	"design_approved":     "performance_checked",
	"performance_checked": "published",
}

type AssetFacts struct {
	Status      string
	Source      string
	Creator     string
	LicenseID   *int
	LicenseURL  string
	AIGenerated bool
	HasFile     bool
	SizeBytes   int64
	Kind        string
}

// CheckTransition enforces the governance gates. Pure so it is unit-tested.
func CheckTransition(a AssetFacts, to string) error {
	if to == "rejected" {
		return nil
	}
	if to == "imported" && a.Status == "rejected" {
		return nil
	}
	if nextStatus[a.Status] != to {
		return fmt.Errorf("cannot move from %s to %s", a.Status, to)
	}
	switch to {
	case "license_verified":
		if a.LicenseID == nil || a.LicenseURL == "" || a.Source == "" || a.Creator == "" {
			return errors.New("source, creator, license and license URL must be recorded")
		}
	case "design_approved":
		if a.AIGenerated {
			return errors.New("AI-generated imagery cannot be approved as a production brand asset (§27.1)")
		}
	case "performance_checked":
		if !a.HasFile {
			return errors.New("upload the optimized file first")
		}
		if b := Budget(a.Kind); a.SizeBytes > b {
			return fmt.Errorf("file is %d KB; budget for %s is %d KB", a.SizeBytes>>10, a.Kind, b>>10)
		}
	}
	return nil
}
