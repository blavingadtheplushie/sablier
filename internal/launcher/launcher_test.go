package launcher

import "testing"

func TestExitCode(t *testing.T) {
	code, err := Run("sh", []string{"-c", "exit 3"}, Options{RunID: "r1"})
	if err != nil || code != 3 {
		t.Fatalf("got %d, %v, want 3", code, err)
	}
}

func TestRunIDIsSet(t *testing.T) {
	code, err := Run("sh", []string{"-c", `test "$SABLIER_RUN_ID" = r1`}, Options{RunID: "r1"})
	if err != nil || code != 0 {
		t.Fatalf("SABLIER_RUN_ID not passed: %d, %v", code, err)
	}
}

func TestNotFound(t *testing.T) {
	if _, err := Run("sablier-no-such-harness", nil, Options{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewRunID(t *testing.T) {
	a, err := NewRunID()
	b, _ := NewRunID()
	if err != nil || len(a) != 32 || a == b {
		t.Fatalf("got %q %q %v", a, b, err)
	}
}
