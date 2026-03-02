package audit

import "testing"

func TestVerifyChainDetectsTamper(t *testing.T) {
	entries := []chainEntry{
		{1, "alice", "login", "ok", "", hashFor("", "alice", "login", "ok")},
	}
	entries = append(entries, chainEntry{2, "bob", "model.deploy", "m1:v1", entries[0].entry, hashFor(entries[0].entry, "bob", "model.deploy", "m1:v1")})

	if ok, _ := verifyChain(entries); !ok {
		t.Fatal("expected valid chain")
	}
	entries[1].details = "m1:v2"
	if ok, msg := verifyChain(entries); ok || msg == "" {
		t.Fatal("expected tampered chain to fail with message")
	}
}
