package media

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

// Runs against a real S3-compatible endpoint when TEST_S3_ENDPOINT is set (SeaweedFS in compose).
func TestS3RoundTrip(t *testing.T) {
	ep := os.Getenv("TEST_S3_ENDPOINT")
	if ep == "" {
		t.Skip("TEST_S3_ENDPOINT not set")
	}
	ctx := context.Background()
	s, err := NewS3(ctx, S3Config{Endpoint: ep, Bucket: "eduplay-test", AccessKey: os.Getenv("TEST_S3_ACCESS_KEY"), SecretKey: os.Getenv("TEST_S3_SECRET_KEY"), Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	n, err := s.Put(ctx, "assets/test.svg", strings.NewReader("<svg/>"))
	if err != nil || n != 6 {
		t.Fatalf("put: %d %v", n, err)
	}
	f, err := s.Open(ctx, "assets/test.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, _ := io.ReadAll(f)
	if string(b) != "<svg/>" {
		t.Fatalf("got %q", b)
	}
	if _, err := s.Open(ctx, "assets/missing.svg"); err == nil {
		t.Fatal("missing object must error")
	}
}
