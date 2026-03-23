package watcher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/codeximport"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	log "github.com/sirupsen/logrus"
)

func (w *Watcher) refreshCodexSyncWatch() {
	if w == nil || w.watcher == nil {
		return
	}

	source, watchDir := w.resolveCodexSyncPaths()

	if prev := strings.TrimSpace(w.codexSyncWatchDir); prev != "" && prev != watchDir {
		_ = w.watcher.Remove(prev)
	}

	w.codexSyncSource = source
	w.codexSyncWatchDir = watchDir

	if watchDir == "" {
		return
	}
	if watchDir == w.normalizeAuthPath(w.authDir) || watchDir == w.normalizeAuthPath(filepath.Dir(w.configPath)) {
		return
	}
	if err := w.watcher.Add(w.codexSyncWatchDir); err != nil {
		log.Debugf("failed to watch codex sync dir %s: %v", w.codexSyncWatchDir, err)
		return
	}
	log.Debugf("watching codex sync source: %s", w.codexSyncSource)
}

func (w *Watcher) resolveCodexSyncPaths() (sourcePath, watchDir string) {
	if w == nil {
		return "", ""
	}
	w.clientsMutex.RLock()
	cfg := w.config
	w.clientsMutex.RUnlock()
	if cfg == nil || !cfg.CodexSync.Enable {
		return "", ""
	}
	resolved, err := util.ResolveAuthDir(cfg.CodexSync.Source)
	if err != nil || strings.TrimSpace(resolved) == "" {
		return "", ""
	}
	normalized := w.normalizeAuthPath(resolved)
	return normalized, w.normalizeAuthPath(filepath.Dir(normalized))
}

func (w *Watcher) syncCodexSource() (string, error) {
	sourcePath, _ := w.resolveCodexSyncPaths()
	if sourcePath == "" {
		return "", fmt.Errorf("codex sync disabled")
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("codex sync source is empty")
	}

	sum := sha256.Sum256(data)
	sourceHash := hex.EncodeToString(sum[:])
	if sourceHash == strings.TrimSpace(w.lastCodexSyncHash) {
		return "", nil
	}

	result, err := codeximport.Convert(data, time.Now().UTC())
	if err != nil {
		return "", err
	}
	body, err := codeximport.MarshalOutput(result)
	if err != nil {
		return "", err
	}

	w.clientsMutex.RLock()
	targetDir := strings.TrimSpace(w.authDir)
	if cfg := w.config; cfg != nil && strings.TrimSpace(cfg.AuthDir) != "" {
		targetDir = strings.TrimSpace(cfg.AuthDir)
	}
	w.clientsMutex.RUnlock()
	targetDir, err = util.ResolveAuthDir(targetDir)
	if err != nil {
		return "", err
	}
	if targetDir == "" {
		return "", fmt.Errorf("auth dir is empty")
	}
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", err
	}

	outputPath := filepath.Join(targetDir, result.FileName)
	payload := append(body, '\n')
	if existing, err := os.ReadFile(outputPath); err == nil && string(existing) == string(payload) {
		w.lastCodexSyncHash = sourceHash
		return outputPath, nil
	}
	if err := os.WriteFile(outputPath, payload, 0o600); err != nil {
		return "", err
	}

	w.lastCodexSyncHash = sourceHash
	return outputPath, nil
}
