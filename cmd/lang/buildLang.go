package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func isPluralKey(key string) bool {
	return key == "zero" || key == "one" || key == "two" || key == "few" || key == "many"
}

func splitKeyToBaseAndPlural(key string) (string, string) {
	segments := strings.Split(key, ".")
	if len(segments) > 1 && isPluralKey(segments[len(segments)-1]) {
		return strings.Join(segments[:len(segments)-1], "."), segments[len(segments)-1]
	}
	return key, "other"
}

func transformJson(input map[string]interface{}) (map[string]interface{}, error) {
	output := make(map[string]interface{})

	for key, value := range input {
		baseKey, pluralKey := splitKeyToBaseAndPlural(key)

		if existing, exists := output[baseKey]; exists {
			existingMap := existing.(map[string]interface{})
			existingMap[pluralKey] = value
		} else {
			output[baseKey] = map[string]interface{}{
				pluralKey: value,
			}
		}
	}

	return output, nil
}

func main() {
	localiseDir := "internal/i18n/localise"
	outputDir := "internal/i18n/lang"

	// Create output directory if it doesn't exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.MkdirAll(outputDir, os.ModePerm)
	}

	// Read the localise directory
	langDirs, err := os.ReadDir(localiseDir)
	if err != nil {
		fmt.Println("Error reading localise directory:", err)
		return
	}

	for _, langDir := range langDirs {
		if langDir.IsDir() {
			lang := langDir.Name()
			mergedData := make(map[string]interface{})

			// Read JSON files in the language directory
			files, err := os.ReadDir(filepath.Join(localiseDir, lang))
			if err != nil {
				fmt.Println("Error reading language directory:", err)
				continue
			}

			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					filePath := filepath.Join(localiseDir, lang, file.Name())
					data, err := os.ReadFile(filePath)
					if err != nil {
						fmt.Println("Error reading file:", err)
						continue
					}

					var jsonData map[string]interface{}
					if err := json.Unmarshal(data, &jsonData); err != nil {
						fmt.Println("Error parsing JSON file:", err)
						continue
					}

					// Merge JSON data
					for key, value := range jsonData {
						mergedData[key] = value
					}
				}
			}

			transformedData, err := transformJson(mergedData)

			// Write merged data to output file
			outputFilePath := filepath.Join(outputDir, lang+".json")
			mergedJSON, err := json.MarshalIndent(transformedData, "", "  ")
			if err != nil {
				fmt.Println("Error marshalling merged JSON:", err)
				continue
			}

			if err := os.WriteFile(outputFilePath, mergedJSON, 0644); err != nil {
				fmt.Println("Error writing merged JSON to file:", err)
				continue
			}
		}
	}
}
