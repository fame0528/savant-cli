package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CompletionItem represents a single completion entry.
type CompletionItem struct {
	Name        string // Display name
	Path        string // Full path
	Description string // Optional description
	Kind        string // "file", "dir", "agent", "tool"
}

// Completions is a popup completion list for @-mentions.
type Completions struct {
	items      []CompletionItem
	filtered   []CompletionItem
	filter     string
	selected   int
	scroll     int
	visible    bool
	maxVisible int
	width      int
}

// NewCompletions creates a new completions popup.
func NewCompletions(width int) *Completions {
	return &Completions{
		maxVisible: 8,
		width:      width,
	}
}

// Show displays the completions popup with the given items.
func (c *Completions) Show(items []CompletionItem) {
	c.items = items
	c.filtered = items
	c.selected = 0
	c.scroll = 0
	c.visible = true
	c.filter = ""
}

// Hide closes the completions popup.
func (c *Completions) Hide() {
	c.visible = false
	c.filter = ""
}

// IsVisible returns whether the popup is shown.
func (c *Completions) IsVisible() bool {
	return c.visible
}

// Filter updates the filter text and refilters items.
func (c *Completions) Filter(query string) {
	c.filter = query
	c.filtered = c.filterItems(query)
	c.selected = 0
	c.scroll = 0
	if len(c.filtered) == 0 {
		c.Hide()
	}
}

// Selected returns the currently selected completion item.
func (c *Completions) Selected() *CompletionItem {
	if !c.visible || len(c.filtered) == 0 || c.selected >= len(c.filtered) {
		return nil
	}
	return &c.filtered[c.selected]
}

// MoveUp moves the selection up.
func (c *Completions) MoveUp() {
	if c.selected > 0 {
		c.selected--
		if c.selected < c.scroll {
			c.scroll = c.selected
		}
	}
}

// MoveDown moves the selection down.
func (c *Completions) MoveDown() {
	if c.selected < len(c.filtered)-1 {
		c.selected++
		if c.selected >= c.scroll+c.maxVisible {
			c.scroll = c.selected - c.maxVisible + 1
		}
	}
}

// Render draws the completions popup.
func (c *Completions) Render(theme *Theme) string {
	if !c.visible || len(c.filtered) == 0 {
		return ""
	}

	var sb strings.Builder

	// Border top
	sb.WriteString(theme.Dialog.Render("╭"+strings.Repeat("─", c.width-2)+"╮") + "\n")

	// Items
	end := c.scroll + c.maxVisible
	if end > len(c.filtered) {
		end = len(c.filtered)
	}

	for i := c.scroll; i < end; i++ {
		item := c.filtered[i]
		icon := c.getIcon(item.Kind)

		// Highlight selected item
		line := icon + " " + item.Name
		if item.Description != "" {
			line += " " + theme.TextDim.Render("— "+item.Description)
		}

		// Pad to width
		for len(line) < c.width-4 {
			line += " "
		}
		if len(line) > c.width-4 {
			line = line[:c.width-4]
		}

		if i == c.selected {
			sb.WriteString(theme.Dialog.Render("│ ") + theme.Info.Bold(true).Render("▸ "+line) + theme.Dialog.Render(" │") + "\n")
		} else {
			sb.WriteString(theme.Dialog.Render("│  "+line+" │") + "\n")
		}
	}

	// Scroll indicator
	if len(c.filtered) > c.maxVisible {
		scrollInfo := theme.TextDim.Render("  ↑↓ scroll")
		sb.WriteString(theme.Dialog.Render("╰"+strings.Repeat("─", c.width-2)+"╯") + " " + scrollInfo + "\n")
	} else {
		sb.WriteString(theme.Dialog.Render("╰"+strings.Repeat("─", c.width-2)+"╯") + "\n")
	}

	return sb.String()
}

func (c *Completions) getIcon(kind string) string {
	switch kind {
	case "file":
		return "📄"
	case "dir":
		return "📁"
	case "agent":
		return "🤖"
	case "tool":
		return "🔧"
	default:
		return "•"
	}
}

func (c *Completions) filterItems(query string) []CompletionItem {
	if query == "" {
		return c.items
	}

	lower := strings.ToLower(query)
	var result []CompletionItem

	for _, item := range c.items {
		name := strings.ToLower(item.Name)
		path := strings.ToLower(item.Path)

		// Tier 1: Exact name match
		if name == lower {
			result = append([]CompletionItem{item}, result...)
			continue
		}

		// Tier 2: Name prefix
		if strings.HasPrefix(name, lower) {
			result = append(result, item)
			continue
		}

		// Tier 3: Name contains
		if strings.Contains(name, lower) {
			result = append(result, item)
			continue
		}

		// Tier 4: Path contains
		if strings.Contains(path, lower) {
			result = append(result, item)
		}
	}

	// Sort: exact prefix first, then alphabetical
	sort.Slice(result, func(i, j int) bool {
		iPrefix := strings.HasPrefix(strings.ToLower(result[i].Name), lower)
		jPrefix := strings.HasPrefix(strings.ToLower(result[j].Name), lower)
		if iPrefix != jPrefix {
			return iPrefix
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// ScanFiles creates completion items from a directory.
func ScanFiles(root string) []CompletionItem {
	var items []CompletionItem
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		kind := "file"
		desc := ""
		if info.IsDir() {
			kind = "dir"
			desc = "directory"
		} else {
			desc = filepath.Ext(name)
		}

		items = append(items, CompletionItem{
			Name:        name,
			Path:        rel,
			Description: desc,
			Kind:        kind,
		})

		// Limit to 500 items
		if len(items) > 500 {
			return filepath.SkipAll
		}

		return nil
	})
	return items
}
