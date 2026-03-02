package marketmath

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func StableFeatureHash(featureMap map[string]float64, orderedKeys []string) (string, error) {
	out := make([][2]interface{}, 0, len(orderedKeys))
	for _, k := range orderedKeys {
		out = append(out, [2]interface{}{k, featureMap[k]})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
