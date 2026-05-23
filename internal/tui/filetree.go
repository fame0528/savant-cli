package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileEntry represents a file or directory in the tree.
type FileEntry struct {
	Name     string
	Path     string
	IsDir    bool
	Children []FileEntry
	Depth    int
}

// FileTree represents a filesystem tree for the sidebar.
type FileTree struct {
	root    string
	entries []FileEntry
	width   int
	scroll  int
	focused bool
}

// NewFileTree creates a new file tree rooted at the given directory.
func NewFileTree(root string, width int) *FileTree {
	ft := &FileTree{
		root:  root,
		width: width,
	}
	ft.Refresh()
	return ft
}

// Refresh rescans the filesystem and rebuilds the tree.
func (ft *FileTree) Refresh() {
	ft.entries = ft.scanDir(ft.root, 0, 3) // max depth 3
}

// scanDir recursively scans a directory up to maxDepth.
func (ft *FileTree) scanDir(dir string, depth, maxDepth int) []FileEntry {
	if depth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	// Sort: directories first, then files, both alphabetical
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	var result []FileEntry
	for _, e := range entries {
		name := e.Name()

		// Skip hidden files/dirs and common large dirs
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
			continue
		}

		fullPath := filepath.Join(dir, name)
		entry := FileEntry{
			Name:  name,
			Path:  fullPath,
			IsDir: e.IsDir(),
			Depth: depth,
		}

		if e.IsDir() {
			entry.Children = ft.scanDir(fullPath, depth+1, maxDepth)
		}

		result = append(result, entry)
	}

	return result
}

// Render renders the file tree as styled strings.
func (ft *FileTree) Render(theme *Theme, width int) string {
	if len(ft.entries) == 0 {
		return theme.TextDim.Render("  No files found.\n")
	}

	var sb strings.Builder
	ft.renderEntries(&sb, ft.entries, theme, width, 0)
	return sb.String()
}

func (ft *FileTree) renderEntries(sb *strings.Builder, entries []FileEntry, theme *Theme, width, startIdx int) int {
	idx := startIdx
	for _, entry := range entries {
		// Skip if scrolled past
		if idx < ft.scroll {
			idx++
			if entry.IsDir {
				idx = ft.renderEntries(sb, entry.Children, theme, width, idx)
			}
			continue
		}

		indent := strings.Repeat("  ", entry.Depth)

		if entry.IsDir {
			// Directory icon
			icon := "📂"
			if len(entry.Children) == 0 {
				icon = "📁"
			}
			dirLine := fmt.Sprintf("%s%s %s", indent, icon, entry.Name)
			sb.WriteString(theme.Info.Render(dirLine) + "\n")
			idx++

			// Render children
			idx = ft.renderEntries(sb, entry.Children, theme, width, idx)
		} else {
			// File icon based on extension
			icon := getFileIcon(entry.Name)
			fileLine := fmt.Sprintf("%s%s %s", indent, icon, entry.Name)

			// Truncate if too long
			if len(fileLine) > width-4 {
				fileLine = fileLine[:width-7] + "..."
			}

			sb.WriteString(theme.TextDim.Render(fileLine) + "\n")
			idx++
		}
	}
	return idx
}

// getFileIcon returns an appropriate icon for a file based on extension.
func getFileIcon(name string) string {
	ext := filepath.Ext(name)
	switch ext {
	case ".go":
		return "🔹"
	case ".js", ".ts", ".jsx", ".tsx":
		return "🔸"
	case ".py":
		return "🐍"
	case ".rs":
		return "🦀"
	case ".md":
		return "📝"
	case ".json", ".yaml", ".yml", ".toml":
		return "📋"
	case ".html", ".css", ".scss":
		return "🌐"
	case ".sh", ".bash", ".zsh":
		return "⚡"
	case ".sql":
		return "🗃️"
	case ".txt":
		return "📄"
	case ".gitignore", ".env":
		return "⚙️"
	default:
		return "📄"
	}
}

// GetPath returns the full path of the entry at the given line number.
func (ft *FileTree) GetPath(lineNum int) string {
	return ft.getPathFromEntries(ft.entries, lineNum, 0)
}

func (ft *FileTree) getPathFromEntries(entries []FileEntry, target, current int) string {
	for _, entry := range entries {
		if current == target {
			return entry.Path
		}
		current++
		if entry.IsDir {
			if result := ft.getPathFromEntries(entry.Children, target, current); result != "" {
				return result
			}
			current += len(entry.Children)
		}
	}
	return ""
}

// Count returns the total number of visible entries.
func (ft *FileTree) Count() int {
	return ft.countEntries(ft.entries)
}

func (ft *FileTree) countEntries(entries []FileEntry) int {
	count := 0
	for _, e := range entries {
		count++
		if e.IsDir {
			count += ft.countEntries(e.Children)
		}
	}
	return count
}
