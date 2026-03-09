package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PluginInstall represents one installation entry from installed_plugins.json.
type PluginInstall struct {
	Scope        string
	InstallPath  string
	Version      string
	InstalledAt  time.Time
	LastUpdated  time.Time
	GitCommitSha string
}

// PluginManifest represents .claude-plugin/plugin.json contents.
type PluginManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Category    string `json:"category"`
	Strict      bool   `json:"strict"`
	Source      string `json:"source"`
	Author      struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
	LspServers map[string]json.RawMessage `json:"lspServers"`
}

// PluginComponent represents a discovered file within a plugin.
type PluginComponent struct {
	Type string // "agent", "hook", "command", "mcp", "lsp"
	Name string
	Path string
	Size int64
}

// Plugin is a fully resolved plugin for display.
type Plugin struct {
	ID          string // "plugin-name@marketplace"
	Name        string // just the plugin name part
	Marketplace string // just the marketplace part
	Install     PluginInstall
	Manifest    *PluginManifest
	Components  []PluginComponent
	Blocked     bool
	BlockReason string
}

// MarketplaceInfo from known_marketplaces.json.
type MarketplaceInfo struct {
	SourceType string // "git" or "github"
	SourceURL  string // URL or "owner/repo"
}

// PluginTree holds all discovered plugins grouped by marketplace.
type PluginTree struct {
	Plugins      []Plugin
	Marketplaces map[string]MarketplaceInfo
}

// ScanPlugins discovers all installed plugins from claudeDir/plugins/.
func ScanPlugins(claudeDir string) (*PluginTree, error) {
	pluginsDir := filepath.Join(claudeDir, "plugins")

	// Parse installed_plugins.json
	installed, err := parseInstalledPlugins(filepath.Join(pluginsDir, "installed_plugins.json"))
	if err != nil {
		return nil, err
	}

	// Parse blocklist
	blocklist := parseBlocklist(filepath.Join(pluginsDir, "blocklist.json"))

	// Parse marketplaces
	marketplaces := parseMarketplaces(filepath.Join(pluginsDir, "known_marketplaces.json"))

	var plugins []Plugin
	for id, installs := range installed {
		if len(installs) == 0 {
			continue
		}
		inst := installs[0] // use first install entry

		name, mkt := splitPluginID(id)

		p := Plugin{
			ID:          id,
			Name:        name,
			Marketplace: mkt,
			Install:     inst,
		}

		// Read manifest
		p.Manifest = readManifest(inst.InstallPath)

		// Scan components
		p.Components = scanComponents(inst.InstallPath, p.Manifest)

		// Check blocklist
		if reason, blocked := blocklist[id]; blocked {
			p.Blocked = true
			p.BlockReason = reason
		}

		plugins = append(plugins, p)
	}

	// Sort by marketplace then name
	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Marketplace != plugins[j].Marketplace {
			return plugins[i].Marketplace < plugins[j].Marketplace
		}
		return plugins[i].Name < plugins[j].Name
	})

	return &PluginTree{
		Plugins:      plugins,
		Marketplaces: marketplaces,
	}, nil
}

// --- internal helpers ---

type rawInstalledPlugins struct {
	Version int                          `json:"version"`
	Plugins map[string][]json.RawMessage `json:"plugins"`
}

type rawInstall struct {
	Scope        string `json:"scope"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha"`
}

func parseInstalledPlugins(path string) (map[string][]PluginInstall, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var raw rawInstalledPlugins
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	result := make(map[string][]PluginInstall, len(raw.Plugins))
	for id, entries := range raw.Plugins {
		var installs []PluginInstall
		for _, entry := range entries {
			var ri rawInstall
			if json.Unmarshal(entry, &ri) != nil {
				continue
			}
			inst := PluginInstall{
				Scope:        ri.Scope,
				InstallPath:  ri.InstallPath,
				Version:      ri.Version,
				GitCommitSha: ri.GitCommitSha,
			}
			inst.InstalledAt, _ = time.Parse(time.RFC3339, ri.InstalledAt)
			inst.LastUpdated, _ = time.Parse(time.RFC3339, ri.LastUpdated)
			installs = append(installs, inst)
		}
		result[id] = installs
	}
	return result, nil
}

func parseBlocklist(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw struct {
		Plugins []struct {
			Plugin string `json:"plugin"`
			Reason string `json:"reason"`
			Text   string `json:"text"`
		} `json:"plugins"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	result := make(map[string]string, len(raw.Plugins))
	for _, p := range raw.Plugins {
		reason := p.Reason
		if p.Text != "" {
			reason += ": " + p.Text
		}
		result[p.Plugin] = reason
	}
	return result
}

func parseMarketplaces(path string) map[string]MarketplaceInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]struct {
		Source struct {
			Source string `json:"source"`
			URL    string `json:"url"`
			Repo   string `json:"repo"`
		} `json:"source"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	result := make(map[string]MarketplaceInfo, len(raw))
	for name, m := range raw {
		info := MarketplaceInfo{SourceType: m.Source.Source}
		if m.Source.URL != "" {
			info.SourceURL = m.Source.URL
		} else if m.Source.Repo != "" {
			info.SourceURL = m.Source.Repo
		}
		result[name] = info
	}
	return result
}

func splitPluginID(id string) (name, marketplace string) {
	if idx := strings.LastIndex(id, "@"); idx > 0 {
		return id[:idx], id[idx+1:]
	}
	return id, ""
}

func readManifest(installPath string) *PluginManifest {
	// Try plugin.json first, then marketplace.json
	for _, name := range []string{"plugin.json", "marketplace.json"} {
		path := filepath.Join(installPath, ".claude-plugin", name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m PluginManifest
		if json.Unmarshal(data, &m) == nil && m.Name != "" {
			return &m
		}
	}
	return nil
}

func scanComponents(installPath string, manifest *PluginManifest) []PluginComponent {
	var components []PluginComponent

	// Standard subdirectories
	subdirs := []struct {
		dir  string
		typ  string
		exts []string
	}{
		{"agents", "agent", []string{".md"}},
		{"hooks", "hook", []string{".py", ".sh", ".bash"}},
		{"commands", "command", []string{".md"}},
		{"mcps", "mcp", []string{".json"}},
	}

	for _, sd := range subdirs {
		dir := filepath.Join(installPath, sd.dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			matched := len(sd.exts) == 0
			for _, x := range sd.exts {
				if ext == x {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			info, _ := e.Info()
			var size int64
			if info != nil {
				size = info.Size()
			}
			components = append(components, PluginComponent{
				Type: sd.typ,
				Name: e.Name(),
				Path: filepath.Join(dir, e.Name()),
				Size: size,
			})
		}
	}

	// LSP servers from manifest
	if manifest != nil && len(manifest.LspServers) > 0 {
		for name := range manifest.LspServers {
			components = append(components, PluginComponent{
				Type: "lsp",
				Name: name,
			})
		}
	}

	return components
}
