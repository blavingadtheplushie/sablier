package launcher

import "testing"

func TestExitCode(t *testing.T) {
	code, err := Run("sh", []string{"-c", "exit 3"})
	if err != nil || code != 3 {
		t.Fatalf("got %d, %v, want 3", code, err)
	}
}

func TestRunIDIsSet(t *testing.T) {
	code, err := Run("sh", []string{"-c", `test ${#SABLIER_RUN_ID} -eq 32`})
	if err != nil || code != 0 {
		t.Fatalf("SABLIER_RUN_ID missing or wrong length: %d, %v", code, err)
	}
}

func TestNotFound(t *testing.T) {
	if _, err := Run("sablier-no-such-harness", nil); err == nil {
		t.Fatal("expected error")
	}
}
