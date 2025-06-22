package processor

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// LinkUpdater handles updating links when files are moved
type LinkUpdater struct {
	parser *LinkParser
}

// NewLinkUpdater creates a new link updater
func NewLinkUpdater() *LinkUpdater {
	return &LinkUpdater{
		parser: NewLinkParser(),
	}
}

// UpdateReferences updates all links in content based on file moves
func (u *LinkUpdater) UpdateReferences(content string, moves []FileMove) string {
	if len(moves) == 0 {
		return content
	}

	// Create a map for quick lookups
	moveMap := u.createMoveMap(moves)

	// Parse all links
	links := u.parser.Extract(content)
	if len(links) == 0 {
		return content
	}

	// Process links in reverse order to avoid position shifts
	result := content
	for i := len(links) - 1; i >= 0; i-- {
		link := links[i]

		// Normalize the link target to match move map keys
		normalizedTarget := u.normalizeLinkTarget(link.Target, link.Type)

		// Check if this link target was moved
		if newPath, moved := moveMap[normalizedTarget]; moved {
			// Update the link
			newLink := u.createUpdatedLink(link, newPath)
			result = result[:link.Position.Start] + newLink + result[link.Position.End:]
		}
	}

	return result
}

// UpdateFile updates all links in a VaultFile
func (u *LinkUpdater) UpdateFile(file *vault.VaultFile, moves []FileMove) bool {
	originalBody := file.Body
	file.Body = u.UpdateReferences(file.Body, moves)

	// Update parsed links if content changed
	if file.Body != originalBody {
		u.parser.UpdateFile(file)
		return true
	}

	return false
}

// UpdateBatch updates links in multiple files and returns list of modified files
func (u *LinkUpdater) UpdateBatch(files []*vault.VaultFile, moves []FileMove) []*vault.VaultFile {
	var modifiedFiles []*vault.VaultFile

	for _, file := range files {
		if u.UpdateFile(file, moves) {
			modifiedFiles = append(modifiedFiles, file)
		}
	}

	return modifiedFiles
}

// createMoveMap creates a map from old paths to new paths
func (u *LinkUpdater) createMoveMap(moves []FileMove) map[string]string {
	moveMap := make(map[string]string)

	for _, move := range moves {
		// Add both with and without .md extension for wiki links
		moveMap[move.From] = move.To

		// For .md files, also add without extension for wiki link matching
		if strings.HasSuffix(move.From, ".md") {
			fromWithoutExt := strings.TrimSuffix(move.From, ".md")
			toWithoutExt := strings.TrimSuffix(move.To, ".md")
			moveMap[fromWithoutExt] = toWithoutExt
		}
		
		// Add URL-encoded versions for matching encoded links (Obsidian-style)
		encodedFrom := u.obsidianURLEncode(move.From)
		if encodedFrom != move.From {
			moveMap[encodedFrom] = move.To
		}
		
		if strings.HasSuffix(move.From, ".md") {
			fromWithoutExt := strings.TrimSuffix(move.From, ".md")
			encodedFromWithoutExt := u.obsidianURLEncode(fromWithoutExt)
			if encodedFromWithoutExt != fromWithoutExt {
				toWithoutExt := strings.TrimSuffix(move.To, ".md")
				moveMap[encodedFromWithoutExt] = toWithoutExt
			}
		}
	}

	return moveMap
}

// normalizeLinkTarget converts a link target to the format used in move maps
func (u *LinkUpdater) normalizeLinkTarget(target string, linkType LinkType) string {
	// URL decode the target first to handle %20, etc.
	decodedTarget, err := url.QueryUnescape(target)
	if err != nil {
		// If decoding fails, use original target
		decodedTarget = target
	}
	
	switch linkType {
	case WikiLink:
		// Wiki links might not have .md extension
		if !strings.HasSuffix(decodedTarget, ".md") && !strings.Contains(filepath.Base(decodedTarget), ".") {
			return decodedTarget + ".md"
		}
		return decodedTarget
	case MarkdownLink, EmbedLink:
		// Markdown and embed links should have their extension preserved
		// Return the decoded version for proper matching
		return decodedTarget
	default:
		return target
	}
}

// createUpdatedLink creates the new link text for a moved file
func (u *LinkUpdater) createUpdatedLink(link Link, newPath string) string {
	switch link.Type {
	case WikiLink:
		// Remove .md extension for wiki links
		newTarget := strings.TrimSuffix(newPath, ".md")

		// Check if the original link text matches the target (simple link)
		originalTarget := link.Target
		decodedOriginal, err := url.QueryUnescape(originalTarget)
		if err != nil {
			decodedOriginal = originalTarget
		}

		if link.Text == originalTarget || link.Text == decodedOriginal {
			// Simple wiki link [[target]]
			return "[[" + newTarget + "]]"
		} else {
			// Wiki link with alias [[target|alias]]
			return "[[" + newTarget + "|" + link.Text + "]]"
		}

	case MarkdownLink:
		// Markdown link [text](target)
		// Obsidian always URL-encodes paths with spaces or special characters
		outputPath := newPath
		if u.needsURLEncoding(newPath) {
			outputPath = u.obsidianURLEncode(newPath)
		}
		return "[" + link.Text + "](" + outputPath + ")"

	case EmbedLink:
		// Embed link ![[target]]
		return "![[" + newPath + "]]"

	default:
		return ""
	}
}

// needsURLEncoding checks if a path needs URL encoding for markdown links
func (u *LinkUpdater) needsURLEncoding(path string) bool {
	// Check for characters that typically need encoding in markdown links
	needsEncoding := strings.ContainsAny(path, " '\"()[]{}#%&+,;=?@")
	return needsEncoding
}

// obsidianURLEncode encodes a path the way Obsidian does for markdown links
// Obsidian only encodes specific characters and leaves paths mostly intact
func (u *LinkUpdater) obsidianURLEncode(path string) string {
	// Replace specific characters that Obsidian encodes
	result := strings.ReplaceAll(path, " ", "%20")
	result = strings.ReplaceAll(result, "'", "%27")
	result = strings.ReplaceAll(result, "\"", "%22")
	result = strings.ReplaceAll(result, "(", "%28")
	result = strings.ReplaceAll(result, ")", "%29")
	result = strings.ReplaceAll(result, "[", "%5B")
	result = strings.ReplaceAll(result, "]", "%5D")
	result = strings.ReplaceAll(result, "{", "%7B")
	result = strings.ReplaceAll(result, "}", "%7D")
	result = strings.ReplaceAll(result, "#", "%23")
	return result
}

// TrackMoves creates a log of file moves for later reference
func (u *LinkUpdater) TrackMoves(moves []FileMove) MoveLog {
	return MoveLog{
		Moves: moves,
	}
}

// MoveLog represents a log of file movements
type MoveLog struct {
	Moves []FileMove
}

// ApplyMoveLog applies a move log to update links in files
func (u *LinkUpdater) ApplyMoveLog(files []*vault.VaultFile, log MoveLog) []*vault.VaultFile {
	return u.UpdateBatch(files, log.Moves)
}
