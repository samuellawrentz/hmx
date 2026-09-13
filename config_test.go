package main

import (
	"os"
	"path/filepath"
	"testing"
)

// flag --key=value > env hmx_key > conf file `key = value` > default; hyphens become underscores
func TestConfig(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "config")
	os.WriteFile(conf, []byte("map_dir = /from/file\nline-spacing=2\n\nbindx = expand_all\nonly_file=f\n"), 0644)
	cfg, file := loadConfig([]string{"--map-dir=/from/flag", "--flag-only", "a.hmm"}, []string{"hmx_line_spacing=3", "OTHER=1"}, conf)
	want := map[string]string{"map_dir": "/from/flag", "line_spacing": "3", "bindx": "expand_all", "only_file": "f", "flag_only": "1"}
	for k, w := range want {
		if cfg.get(k, "default") != w {
			t.Errorf("%s = %q, want %q", k, cfg.get(k, "default"), w)
		}
	}
	if cfg.get("missing", "default") != "default" || file != "a.hmm" {
		t.Errorf("default=%q file=%q", cfg.get("missing", "default"), file)
	}
	cfg, file = loadConfig(nil, nil, filepath.Join(t.TempDir(), "none"))
	if len(cfg) != 0 || file != "" {
		t.Errorf("no config: %v %q", cfg, file)
	}
}
