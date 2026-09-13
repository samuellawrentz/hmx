package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type App struct {
	s         tcell.Screen
	m         *Map
	cfg       Config
	mapDir    string
	msg       string
	query     string
	listQuery string
	clip      string
	stack     []NavEntry
	quit      bool
}

type Config map[string]string

func (c Config) get(key, def string) string {
	if v, ok := c[key]; ok {
		return v
	}
	return def
}

func normKey(k string) string {
	return strings.ReplaceAll(strings.TrimSpace(k), "-", "_")
}

// loadConfig ports the config parsing at ref/hmx.php lines 27-80: flag > env > file > default.
func loadConfig(argv []string, env []string, conf string) (Config, string) {
	cfg := Config{}
	file := ""

	if b, err := os.ReadFile(conf); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			val := ""
			if len(parts) > 1 {
				val = strings.TrimSpace(parts[1])
			}
			cfg[normKey(parts[0])] = val
		}
	}

	for _, e := range env {
		if !strings.HasPrefix(e, "hmx_") {
			continue
		}
		kv := strings.SplitN(e[len("hmx_"):], "=", 2)
		val := ""
		if len(kv) > 1 {
			val = kv[1]
		}
		cfg[normKey(kv[0])] = val
	}

	for _, a := range argv {
		if strings.HasPrefix(a, "--") {
			kv := strings.SplitN(a[2:], "=", 2)
			val := "1"
			if len(kv) > 1 {
				val = strings.Trim(kv[1], `"`)
			}
			cfg[normKey(kv[0])] = val
		} else if file == "" {
			file = a
		}
	}

	return cfg, file
}

func confPath(argv []string) string {
	for _, a := range argv {
		if v, ok := strings.CutPrefix(a, "--config="); ok {
			return v
		}
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "hmx", "config")
}

// openMap mirrors load_map: Load, collapse_all, collapse_level(1), build, centre.
func (a *App) openMap(file string) {
	m, _ := Load(file)
	a.m = m
	a.m.loadTasks()
	a.m.CollapseAll()
	a.m.CollapseLevel(1)
	a.build()
	a.m.Center(a.s.Size())
}

func main() {
	argv := os.Args[1:]
	cfg, file := loadConfig(argv, os.Environ(), confPath(argv))

	for k, v := range cfg {
		if !strings.HasPrefix(k, "bind") {
			continue
		}
		if _, ok := actions[v]; !ok {
			fmt.Fprintf(os.Stderr, "Config error! %q is an unknown command.\n", v)
			os.Exit(1)
		}
		mapKeys[strings.TrimSpace(k[len("bind"):])] = v
	}

	home, _ := os.UserHomeDir()
	mapDir := cfg.get("map_dir", filepath.Join(home, "maps"))
	if err := os.MkdirAll(mapDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	s, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := s.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer s.Fini()

	a := &App{s: s, cfg: cfg, mapDir: mapDir}

	for {
		if file == "" {
			file = a.listScreen()
			if file == "" {
				break
			}
		}
		a.openMap(file)
		if a.listQuery != "" {
			a.query = a.listQuery
			a.listQuery = ""
			nextSearchResult(a)
		}
		a.quit = false
		a.draw()

		for !a.quit {
			switch ev := s.PollEvent().(type) {
			case *tcell.EventResize:
				a.draw()
			case *tcell.EventKey:
				a.msg = ""
				if fn, ok := actions[mapKeys[keyName(ev)]]; ok {
					fn(a)
				}
				a.draw()
			}
		}
		file = ""
	}
}
