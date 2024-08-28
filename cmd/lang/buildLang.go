package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

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
					data, err := ioutil.ReadFile(filePath)
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

			// Write merged data to output file
			outputFilePath := filepath.Join(outputDir, lang+".json")
			mergedJSON, err := json.MarshalIndent(mergedData, "", "  ")
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
