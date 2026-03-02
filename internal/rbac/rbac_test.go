package rbac

import "testing"

func TestAllowed(t *testing.T) {
	if !Allowed("admin", "anything") {
		t.Fatal("admin should be allowed")
	}
	if !Allowed("admin", "governance:read") {
		t.Fatal("admin should read governance")
	}
	if !Allowed("analyst", "alerts:write") {
		t.Fatal("analyst should ack alerts")
	}
	if !Allowed("analyst", "replay:write") {
		t.Fatal("analyst should trigger replay")
	}
	if Allowed("analyst", "model:deploy") {
		t.Fatal("analyst must not deploy models")
	}
	if Allowed("viewer", "alerts:write") {
		t.Fatal("viewer must not acknowledge alerts")
	}
	if !Allowed("viewer", "read") {
		t.Fatal("viewer should be read-only")
	}
	if Allowed("viewer", "replay:write") {
		t.Fatal("viewer must not trigger replay")
	}
}

func TestGovernanceReadRoles(t *testing.T) {
	if Allowed("viewer", "governance:read") {
		t.Fatal("viewer must not read governance")
	}
	if Allowed("analyst", "governance:read") {
		t.Fatal("analyst must not read governance")
	}
	if !Allowed("admin", "governance:read") {
		t.Fatal("admin should read governance")
	}
}
