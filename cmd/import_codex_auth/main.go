package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/codeximport"
)

func main() {
	defaultInput, err := os.UserHomeDir()
	if err != nil {
		exitf("resolve home directory: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		exitf("resolve working directory: %v", err)
	}
	defaultOutDir := resolveDefaultOutputDir(defaultInput, cwd, func(path string) error {
		_, err := os.Stat(path)
		return err
	})

	input := flag.String("input", filepath.Join(defaultInput, ".codex", "auth.json"), "path to ~/.codex/auth.json")
	outDir := flag.String("out-dir", defaultOutDir, "directory to write the converted CLIProxyAPI auth file into")
	flag.Parse()

	raw, err := os.ReadFile(*input)
	if err != nil {
		exitf("read input file: %v", err)
	}

	result, err := codeximport.Convert(raw, timeNowUTC())
	if err != nil {
		exitf("convert auth file: %v", err)
	}

	body, err := codeximport.MarshalOutput(result)
	if err != nil {
		exitf("marshal output: %v", err)
	}

	if err := os.MkdirAll(*outDir, 0o700); err != nil {
		exitf("create output directory: %v", err)
	}

	outputPath := filepath.Join(*outDir, result.FileName)
	if err := os.WriteFile(outputPath, append(body, '\n'), 0o600); err != nil {
		exitf("write output file: %v", err)
	}

	fmt.Println(outputPath)
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}

func resolveDefaultOutputDir(homeDir, cwd string, statFn func(string) error) string {
	homeDir = filepath.Clean(homeDir)
	cwd = filepath.Clean(cwd)

	if homeDir != "" {
		serviceDir := filepath.Join(homeDir, ".cli-proxy-api")
		if statFn != nil && statFn(serviceDir) == nil {
			return serviceDir
		}
	}

	return filepath.Join(cwd, "auths")
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
