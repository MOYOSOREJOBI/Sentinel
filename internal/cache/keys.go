package cache

import "fmt"

func ReadModelPrefixes() []string {
	return []string{
		"queue:v3:", "cc:v3:", "trust:v3:", "worldmap:v3:", "exec:v3:",
		"queue:v2:", "cc:v2:", "trust:v2:", "worldmap:v2:", "exec:v2:",
		"queue:v1:", "cc:v1:", "trust:v1:", "worldmap:v1:", "exec:v1:",
	}
}

func QueueKey(role, region, country, sector, industry, venue, symbol, locale, window string) string {
	return fmt.Sprintf("queue:v3:%s:%s:%s:%s:%s:%s:%s:%s:%s", role, region, country, sector, industry, venue, symbol, locale, window)
}

func CommandCenterKey(role, region, country, sector, industry, venue, symbol, locale, window string) string {
	return fmt.Sprintf("cc:v3:%s:%s:%s:%s:%s:%s:%s:%s:%s", role, region, country, sector, industry, venue, symbol, locale, window)
}

func TrustKey(region, country, sector, industry, venue, symbol, locale, window string) string {
	return fmt.Sprintf("trust:v3:%s:%s:%s:%s:%s:%s:%s:%s", region, country, sector, industry, venue, symbol, locale, window)
}

func WorldMapKey(region, country, sector, industry, venue, symbol, locale, window string) string {
	return fmt.Sprintf("worldmap:v3:%s:%s:%s:%s:%s:%s:%s:%s", region, country, sector, industry, venue, symbol, locale, window)
}

func ExecutiveKey(region, country, sector, industry, venue, symbol, locale, window string) string {
	return fmt.Sprintf("exec:v3:%s:%s:%s:%s:%s:%s:%s:%s", region, country, sector, industry, venue, symbol, locale, window)
}
