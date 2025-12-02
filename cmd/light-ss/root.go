package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "light-ss",
	Short: "A lightweight shadowsocks client with local proxy support",
	Long: `light-ss is a simple shadowsocks client that provides local HTTP/HTTPS and SOCKS5
proxy servers. It forwards all traffic through a shadowsocks server for secure browsing.`,
	Version: "0.1.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(testCmd)
}
