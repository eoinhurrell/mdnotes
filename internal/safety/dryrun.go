package safety

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Operation represents an operation that would be performed
type Operation struct {
	Type              string        `json:"type"`
	File              string        `json:"file"`
	Changes           []Change      `json:"changes"`
	Description       string        `json:"description"`
	EstimatedDuration time.Duration `json:"estimated_duration,omitempty"`
	Timestamp         time.Time     `json:"timestamp"`
}

// Change represents a single change within an operation
type Change struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
	Action   string      `json:"action"` // add, modify, remove, cast, etc.
	Reason   string      `json:"reason,omitempty"`
}

// String returns a human-readable description of the change
func (c Change) String() string {
	switch c.Action {
	case "add":
		return fmt.Sprintf("add field '%s'", c.Field)
	case "modify":
		return fmt.Sprintf("modify field '%s' from '%v' to '%v'", c.Field, c.OldValue, c.NewValue)
	case "remove":
		return fmt.Sprintf("remove field '%s'", c.Field)
	case "cast":
		oldType := getTypeName(c.OldValue)
		newType := getTypeName(c.NewValue)
		return fmt.Sprintf("cast field '%s' from %s to %s", c.Field, oldType, newType)
	default:
		return fmt.Sprintf("%s field '%s'", c.Action, c.Field)
	}
}

// getTypeName returns a human-readable type name
func getTypeName(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case int, int64, float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}, []string:
		return "array"
	case time.Time:
		return "date"
	default:
		return "object"
	}
}

// SummaryStats provides summary statistics about operations
type SummaryStats struct {
	TotalOperations int            `json:"total_operations"`
	FilesAffected   int            `json:"files_affected"`
	TotalChanges    int            `json:"total_changes"`
	OperationTypes  map[string]int `json:"operation_types"`
	ChangeTypes     map[string]int `json:"change_types"`
	EstimatedTime   time.Duration  `json:"estimated_time"`
}

// DryRunRecorder records operations without executing them
type DryRunRecorder struct {
	operations []Operation
	mutex      sync.RWMutex
}

// NewDryRunRecorder creates a new dry-run recorder
func NewDryRunRecorder() *DryRunRecorder {
	return &DryRunRecorder{
		operations: make([]Operation, 0),
	}
}

// Record adds an operation to the dry-run log
func (d *DryRunRecorder) Record(operation Operation) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Set timestamp if not already set
	if operation.Timestamp.IsZero() {
		operation.Timestamp = time.Now()
	}

	d.operations = append(d.operations, operation)
}

// OperationCount returns the number of recorded operations
func (d *DryRunRecorder) OperationCount() int {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return len(d.operations)
}

// GetOperations returns all recorded operations
func (d *DryRunRecorder) GetOperations() []Operation {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	// Return a copy to prevent modification
	result := make([]Operation, len(d.operations))
	copy(result, d.operations)
	return result
}

// GetOperationsForFile returns operations for a specific file
func (d *DryRunRecorder) GetOperationsForFile(filename string) []Operation {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var result []Operation
	for _, op := range d.operations {
		if op.File == filename {
			result = append(result, op)
		}
	}
	return result
}

// GetOperationsByType returns operations of a specific type
func (d *DryRunRecorder) GetOperationsByType(opType string) []Operation {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var result []Operation
	for _, op := range d.operations {
		if op.Type == opType {
			result = append(result, op)
		}
	}
	return result
}

// Clear removes all recorded operations
func (d *DryRunRecorder) Clear() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.operations = d.operations[:0]
}

// GenerateReport generates a human-readable report
func (d *DryRunRecorder) GenerateReport() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if len(d.operations) == 0 {
		return "No operations would be performed."
	}

	var report strings.Builder
	report.WriteString(fmt.Sprintf("Dry Run Report - %d operations would be performed:\n\n", len(d.operations)))

	// Group operations by file
	fileOps := make(map[string][]Operation)
	for _, op := range d.operations {
		fileOps[op.File] = append(fileOps[op.File], op)
	}

	// Generate report for each file
	for file, ops := range fileOps {
		report.WriteString(fmt.Sprintf("📄 %s:\n", file))
		for _, op := range ops {
			report.WriteString(fmt.Sprintf("  • %s (%s)\n", op.Description, op.Type))
			for _, change := range op.Changes {
				report.WriteString(fmt.Sprintf("    - Would %s\n", change.String()))
			}
		}
		report.WriteString("\n")
	}

	// Add summary
	stats := d.getSummaryStatsUnsafe()
	report.WriteString("📊 Summary:\n")
	report.WriteString(fmt.Sprintf("  • Total operations: %d\n", stats.TotalOperations))
	report.WriteString(fmt.Sprintf("  • Files affected: %d\n", stats.FilesAffected))
	report.WriteString(fmt.Sprintf("  • Total changes: %d\n", stats.TotalChanges))

	if stats.EstimatedTime > 0 {
		report.WriteString(fmt.Sprintf("  • Estimated time: %v\n", stats.EstimatedTime))
	}

	report.WriteString("\n💡 Use --dry-run=false to apply these changes.\n")

	return report.String()
}

// GenerateJSONReport generates a JSON report
func (d *DryRunRecorder) GenerateJSONReport() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	report := struct {
		Operations  []Operation  `json:"operations"`
		Summary     SummaryStats `json:"summary"`
		GeneratedAt time.Time    `json:"generated_at"`
	}{
		Operations:  d.operations,
		Summary:     d.getSummaryStatsUnsafe(),
		GeneratedAt: time.Now(),
	}

	jsonData, _ := json.MarshalIndent(report, "", "  ")
	return string(jsonData)
}

// GetSummaryStats returns summary statistics
func (d *DryRunRecorder) GetSummaryStats() SummaryStats {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.getSummaryStatsUnsafe()
}

// getSummaryStatsUnsafe gets summary stats without locking (internal use)
func (d *DryRunRecorder) getSummaryStatsUnsafe() SummaryStats {
	stats := SummaryStats{
		TotalOperations: len(d.operations),
		OperationTypes:  make(map[string]int),
		ChangeTypes:     make(map[string]int),
	}

	filesAffected := make(map[string]bool)
	totalChanges := 0
	var totalTime time.Duration

	for _, op := range d.operations {
		// Count operation types
		stats.OperationTypes[op.Type]++

		// Track affected files
		filesAffected[op.File] = true

		// Count changes and change types
		totalChanges += len(op.Changes)
		for _, change := range op.Changes {
			stats.ChangeTypes[change.Action]++
		}

		// Sum estimated time
		totalTime += op.EstimatedDuration
	}

	stats.FilesAffected = len(filesAffected)
	stats.TotalChanges = totalChanges
	stats.EstimatedTime = totalTime

	return stats
}
