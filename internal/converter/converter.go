package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/xrdavies/light-ss/internal/config"
)

// Convert converts a config file from one format to another
func Convert(fromFormat, inputPath, outputPath string) error {
	var cfg *config.Config
	var err error

	// Parse input based on format
	switch fromFormat {
	case "ss-local", "shadowsocks-libev":
		cfg, err = FromSSLocal(inputPath)
	case "clash":
		cfg, err = FromClash(inputPath)
	default:
		return fmt.Errorf("unsupported format: %s (supported: ss-local, clash)", fromFormat)
	}

	if err != nil {
		return err
	}

	// Determine output format from extension
	ext := strings.ToLower(filepath.Ext(outputPath))
	var data []byte

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(cfg, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cfg)
	default:
		// Default to JSON if no extension
		data, err = json.MarshalIndent(cfg, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write output
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// PrintConfig prints a config in JSON format to stdout
func PrintConfig(cfg *config.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
