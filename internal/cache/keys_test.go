package cache

import "testing"

func TestKeysIncludeLocaleDimension(t *testing.T) {
	a := QueueKey("viewer", "NA", "US", "Tech", "Software", "XNAS", "AAPL", "en", "24h")
	b := QueueKey("viewer", "NA", "US", "Tech", "Software", "XNAS", "AAPL", "fr", "24h")
	if a == b {
		t.Fatal("expected different keys by locale")
	}
}

func TestReadModelPrefixesIncludesCurrentGeneration(t *testing.T) {
	p := ReadModelPrefixes()
	want := map[string]bool{"queue:v3:": false, "cc:v3:": false, "trust:v3:": false, "worldmap:v3:": false, "exec:v3:": false}
	for _, v := range p {
		if _, ok := want[v]; ok {
			want[v] = true
		}
	}
	for k, found := range want {
		if !found {
			t.Fatalf("missing prefix %s", k)
		}
	}
}
