package query

import (
	"context"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func buildCommandCenterSummary(window string, rows []QueueRow) map[string]any {
	open := len(rows)
	high := 0

	topIncidents := make([]map[string]any, 0, 5)
	severity := map[string]int{"critical": 0, "high": 0, "elevated": 0, "stable": 0}
	histogram := []int{0, 0, 0, 0, 0}

	type bucketCount struct {
		Key   string
		Label string
		Count int
	}
	countryCounts := map[string]bucketCount{}
	industryCounts := map[string]int{}

	for idx, row := range rows {
		sev := strings.ToLower(strings.TrimSpace(row.SeverityBand))
		switch {
		case strings.Contains(sev, "critical"):
			severity["critical"]++
			high++
		case strings.Contains(sev, "high"):
			severity["high"]++
			high++
		case strings.Contains(sev, "elevated"):
			severity["elevated"]++
		default:
			severity["stable"]++
		}

		switch {
		case row.CompositeRisk < 0.2:
			histogram[0]++
		case row.CompositeRisk < 0.4:
			histogram[1]++
		case row.CompositeRisk < 0.6:
			histogram[2]++
		case row.CompositeRisk < 0.8:
			histogram[3]++
		default:
			histogram[4]++
		}

		code := strings.ToUpper(strings.TrimSpace(row.CountryISO2))
		if code != "" && code != "XX" {
			item := countryCounts[code]
			item.Key = code
			item.Label = strings.TrimSpace(row.CountryName)
			item.Count++
			countryCounts[code] = item
		}

		industry := strings.TrimSpace(row.Industry)
		if industry == "" {
			industry = "Unknown"
		}
		industryCounts[industry]++

		if idx < 5 {
			topIncidents = append(topIncidents, map[string]any{
				"id":            row.ID,
				"symbol":        row.Symbol,
				"priorityScore": row.PriorityScore,
				"compositeRisk": row.CompositeRisk,
				"severityBand":  row.SeverityBand,
			})
		}
	}

	rankedCountries := make([]bucketCount, 0, len(countryCounts))
	for _, item := range countryCounts {
		rankedCountries = append(rankedCountries, item)
	}
	sort.Slice(rankedCountries, func(i, j int) bool {
		if rankedCountries[i].Count == rankedCountries[j].Count {
			return rankedCountries[i].Key < rankedCountries[j].Key
		}
		return rankedCountries[i].Count > rankedCountries[j].Count
	})

	topCountries := make([]map[string]any, 0, minInt(8, len(rankedCountries)))
	for _, item := range rankedCountries {
		label := item.Label
		if strings.TrimSpace(label) == "" {
			label = item.Key
		}
		topCountries = append(topCountries, map[string]any{
			"countryIso2":   item.Key,
			"countryCode":   item.Key,
			"countryName":   label,
			"incidentCount": item.Count,
		})
		if len(topCountries) == 8 {
			break
		}
	}

	rankedIndustries := make([]bucketCount, 0, len(industryCounts))
	for key, count := range industryCounts {
		rankedIndustries = append(rankedIndustries, bucketCount{Key: key, Count: count})
	}
	sort.Slice(rankedIndustries, func(i, j int) bool {
		if rankedIndustries[i].Count == rankedIndustries[j].Count {
			return rankedIndustries[i].Key < rankedIndustries[j].Key
		}
		return rankedIndustries[i].Count > rankedIndustries[j].Count
	})

	topIndustries := make([]map[string]any, 0, minInt(8, len(rankedIndustries)))
	for _, item := range rankedIndustries {
		topIndustries = append(topIndustries, map[string]any{
			"industry":      item.Key,
			"incidentCount": item.Count,
		})
		if len(topIndustries) == 8 {
			break
		}
	}

	return map[string]any{
		"timeWindow":         window,
		"openIncidents":      open,
		"highRiskCount":      high,
		"backlogDelta":       high - open,
		"topIncidents":       topIncidents,
		"topCountries":       topCountries,
		"topIndustries":      topIndustries,
		"severityBreakdown":  severity,
		"compositeHistogram": histogram,
		"trust":              map[string]any{"state": "stable"},
	}
}

func LoadCommandCenter(ctx context.Context, db *pgxpool.Pool, f QueueFilters) (map[string]any, error) {
	rows, err := LoadQueue(ctx, db, f)
	if err != nil {
		return nil, err
	}
	return buildCommandCenterSummary(f.Window, rows), nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
