package lang

import (
	"embed"
	"encoding/csv"
	"fmt"
	"path"
	"strings"
)

// csvDir holds the per-language CSV files. The language package owns this data
// so it stays free of any Ebiten dependency and can be unit-tested headless.
//
//go:embed csv/*.csv
var csvDir embed.FS

// LoadTemplates parses every embedded language CSV and returns a map of
// language name -> (key -> template text). The language name is the CSV file
// name without its extension (e.g. "english").
func LoadTemplates() map[string]map[string]string {
	const csvDirPath = "csv"

	entries, err := csvDir.ReadDir(csvDirPath)
	if err != nil {
		panic(fmt.Sprintf("cannot open language directory: %v", err))
	}

	templates := make(map[string]map[string]string, len(entries))
	for _, e := range entries {
		data, err := loadCSV(path.Join(csvDirPath, e.Name()))
		if err != nil {
			panic(fmt.Sprintf("cannot load templates for %s: %v", e.Name(), err))
		}
		templates[strings.TrimSuffix(e.Name(), ".csv")] = data
	}

	return templates
}

func loadCSV(filepath string) (map[string]string, error) {
	f, err := csvDir.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m := make(map[string]string)
	r := csv.NewReader(f)
	r.Comment = '#'
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	allContents, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	for i, line := range allContents {
		if len(line) != 2 {
			return nil, fmt.Errorf("line %d length should 2 but %d, data: %v", i, len(line), line)
		}
		m[line[0]] = strings.ReplaceAll(line[1], `\n`, "\n")
	}

	return m, nil
}
