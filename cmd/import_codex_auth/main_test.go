package main

import (
	"errors"
	"testing"
)

func TestResolveDefaultOutputDirPrefersCLIProxyDirWhenPresent(t *testing.T) {
	got := resolveDefaultOutputDir("/home/tester", "/work/project", func(path string) error {
		if path == "/home/tester/.cli-proxy-api" {
			return nil
		}
		return errors.New("missing")
	})

	if want := "/home/tester/.cli-proxy-api"; got != want {
		t.Fatalf("resolveDefaultOutputDir() = %q, want %q", got, want)
	}
}

func TestResolveDefaultOutputDirFallsBackToRepoAuths(t *testing.T) {
	got := resolveDefaultOutputDir("/home/tester", "/work/project", func(string) error {
		return errors.New("missing")
	})

	if want := "/work/project/auths"; got != want {
		t.Fatalf("resolveDefaultOutputDir() = %q, want %q", got, want)
	}
}
