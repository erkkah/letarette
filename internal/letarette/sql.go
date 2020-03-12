package letarette

import (
	"fmt"
	"strings"
)

var sqlCache = map[string]string{}

func loadSearchQuery(strategy int) (string, error) {
	return SQL(fmt.Sprintf("search_%d.sql", strategy))
}

// SQL loads sql code from resources and strips away comments
func SQL(path string) (string, error) {
	if loaded, found := sqlCache[path]; found {
		return loaded, nil
	}

	path = strings.TrimLeft(path, "/")
	sqlAsset := fmt.Sprintf("sql/%s", path)
	sql, err := Asset(sqlAsset)
	if err != nil {
		return "", err
	}
	// Strip comments to avoid name binding getting caught on the url
	// in the license header (!)
	lines := strings.Split(string(sql), "\n")
	uncommented := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		uncommented = append(uncommented, trimmed)
	}
	result := strings.Join(uncommented, "\n")
	sqlCache[path] = result
	return result, nil
}
