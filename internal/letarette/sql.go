// Copyright 2020 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
