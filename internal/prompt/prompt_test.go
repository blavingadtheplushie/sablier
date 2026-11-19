package prompt

import "testing"

func TestApprovalLike(t *testing.T) {
	cases := map[string]bool{
		"yes":                  true,
		"  Yes!! ":             true,
		"ok 👍":                 true,
		"Go ahead.":            true,
		"continue":             true,
		"yes but use postgres": false,
		"fix the failing test": false,
		"no":                   false,
		"":                     false,
		"continue with step 3": false,
	}
	for in, want := range cases {
		if got := ApprovalLike(in); got != want {
			t.Errorf("ApprovalLike(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestHash(t *testing.T) {
	salt := []byte("salt-a")

	if Hash(salt, "Continue ") != Hash(salt, "continue") {
		t.Error("normalized prompts should hash the same")
	}
	if Hash(salt, "yes") == Hash(salt, "no") {
		t.Error("different prompts should differ")
	}
	if Hash(salt, "yes") == Hash([]byte("salt-b"), "yes") {
		t.Error("different salts should differ")
	}
	if len(Hash(salt, "yes")) != 32 {
		t.Error("expected 128-bit hex hash")
	}
}
