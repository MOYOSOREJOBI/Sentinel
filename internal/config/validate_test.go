package config

import "testing"

func TestValidateConfig(t *testing.T) {
	valid := Config{ServiceName: "gateway", HTTPAddr: ":8080", PostgresURL: "postgres://a:b@localhost:5432/db", KafkaBrokers: []string{"localhost:9092"}}
	if err := Validate(valid); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
	invalid := valid
	invalid.PostgresURL = "::bad-url"
	if err := Validate(invalid); err == nil {
		t.Fatal("expected invalid postgres url error")
	}
}
