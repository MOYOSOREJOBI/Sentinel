package query

import "testing"

func TestCountryAggShape(t *testing.T) {
	c := CountryAgg{CountryCode: "US", IncidentCount: 1}
	if c.CountryCode == "" || c.IncidentCount != 1 {
		t.Fatal("bad shape")
	}
}
