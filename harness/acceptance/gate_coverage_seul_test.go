package acceptance

import (
	"strings"
	"testing"
)

// gateMode runs `harness gate <mode>` and returns its exit code and stderr.
func gateMode(t *testing.T, mode string) (int, string) {
	t.Helper()
	code, _, stderr := runBoth(t, t.TempDir(), nil, harness, "gate", mode)
	return code, stderr
}

func TestMUT001_RemovedGateSubcommandsAreUnknown(t *testing.T) {
	wantCode, wantErr := gateMode(t, "nosuchmode")
	if wantCode == 0 || !strings.Contains(wantErr, "unknown mode") {
		t.Fatalf("reference unknown subcommand: code %d, stderr %q", wantCode, wantErr)
	}
	for _, mode := range []string{"red", "green", "characterize", "replay"} {
		t.Run(mode, func(t *testing.T) {
			code, stderr := gateMode(t, mode)
			if code != wantCode {
				t.Errorf("gate %s: exit %d, want %d", mode, code, wantCode)
			}
			if !strings.Contains(stderr, "unknown mode") {
				t.Errorf("gate %s: stderr does not say unknown mode: %q", mode, stderr)
			}
		})
	}
}

func TestMUT001_CoverageSubcommandRemains(t *testing.T) {
	_, stderr := gateMode(t, "coverage")
	if strings.Contains(stderr, "unknown mode") {
		t.Errorf("gate coverage answered as unknown: %q", stderr)
	}
}
