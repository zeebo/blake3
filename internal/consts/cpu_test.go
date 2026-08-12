package consts

import (
	"os"
	"strings"
	"testing"
)

func TestRequireAssembly(t *testing.T) {
	require := os.Getenv("BLAKE3_TEST_REQUIRE_ASM")
	if require == "" {
		t.SkipNow()
	}

	selected := map[string]bool{"avx512": HasAVX512, "avx2": HasAVX2, "sse41": HasSSE41, "sve2": HasSVE2, "neon": HasNEON}
	for _, name := range strings.Split(require, ",") {
		switch ok, known := selected[name]; {
		case !known:
			t.Errorf("unknown backend %q", name)
		case !ok:
			t.Errorf("%s backend not selected", name)
		}
	}
}
