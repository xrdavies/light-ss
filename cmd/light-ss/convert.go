package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xrdavies/light-ss/internal/config"
	"github.com/xrdavies/light-ss/internal/converter"
)

var (
	convertFrom   string
	convertInput  string
	convertOutput string
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert config from other formats",
	Long: `Convert configuration files from other shadowsocks clients to light-ss format.

Supported formats:
  - ss-local (shadowsocks-libev)
  - clash

Examples:
  # Convert ss-local config to JSON
  light-ss convert --from ss-local --input ss-local.json --output config.json

  # Convert Clash config to YAML
  light-ss convert --from clash --input clash.yaml --output config.yaml

  # Print to stdout (default JSON)
  light-ss convert --from ss-local --input ss-local.json`,
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVar(&convertFrom, "from", "", "Source format: ss-local, clash (required)")
	convertCmd.Flags().StringVarP(&convertInput, "input", "i", "", "Input config file (required)")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "Output file (prints to stdout if not specified)")
	convertCmd.MarkFlagRequired("from")
	convertCmd.MarkFlagRequired("input")

	rootCmd.AddCommand(convertCmd)
}

func runConvert(cmd *cobra.Command, args []string) error {
	if convertOutput == "" {
		// Print to stdout
		var cfg *config.Config
		var err error

		switch convertFrom {
		case "ss-local", "shadowsocks-libev":
			cfg, err = converter.FromSSLocal(convertInput)
		case "clash":
			cfg, err = converter.FromClash(convertInput)
		default:
			return fmt.Errorf("unsupported format: %s", convertFrom)
		}

		if err != nil {
			return err
		}

		return converter.PrintConfig(cfg)
	}

	// Convert and write to file
	if err := converter.Convert(convertFrom, convertInput, convertOutput); err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	fmt.Printf("Successfully converted %s to %s\n", convertInput, convertOutput)
	return nil
}
