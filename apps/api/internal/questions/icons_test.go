package questions

import (
	"os"
	"regexp"
	"testing"
)

// Every approved icon must be renderable by the web client.
func TestWebRendersEveryIcon(t *testing.T) {
	src, err := os.ReadFile("../../../web/src/lib/figures.ts")
	if err != nil {
		t.Skip("web source not available")
	}
	for _, name := range Icons {
		if !regexp.MustCompile(`'` + regexp.QuoteMeta(name) + `'\s*:`).Match(src) {
			t.Errorf("icon %q missing from apps/web/src/lib/figures.ts", name)
		}
	}
}
