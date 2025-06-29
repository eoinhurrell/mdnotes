package processor

import (
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

	// Parse all links
	links := u.parser.Extract(content)
	if len(links) == 0 {
		return content
	}

	// Process links in reverse order to avoid position shifts
	result := content
	for i := len(links) - 1; i >= 0; i-- {
		link := links[i]

		// Check if this link should be updated
		for _, move := range moves {
			if link.ShouldUpdate(move.From, move.To) {
				// Update the link
				newLink := link.GenerateUpdatedLink(move.To)
				result = result[:link.Position.Start] + newLink + result[link.Position.End:]
				break // Only apply the first matching move
			}
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

// createUpdatedLink creates the new link text for a moved file
func (u *LinkUpdater) createUpdatedLink(link vault.Link, newPath string) string {
	// Use the GenerateUpdatedLink method from the Link struct
	return link.GenerateUpdatedLink(newPath)
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
