package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ArtifactDeclaration describes a configured artifact path that a step should
// expose after execution.
type ArtifactDeclaration struct {
	Name        string
	Path        string
	Description string
	Optional    bool
}

// ArtifactPath is a normalized path view that keeps both the original input
// and the resolved workspace-relative location.
type ArtifactPath struct {
	Input    string
	Absolute string
	Relative string
}

// Display returns the most compact path representation available.
func (p ArtifactPath) Display() string {
	if p.Relative != "" {
		return p.Relative
	}
	return p.Absolute
}

// ArtifactRecord is the normalized view of one artifact, whether it was
// declared explicitly or discovered after a step finished.
type ArtifactRecord struct {
	Name        string
	Description string
	Path        ArtifactPath
	Declared    bool
	Discovered  bool
	Optional    bool
	Present     bool
	SizeBytes   int64
	ModTime     time.Time
	Problem     string
}

// SourceLabel returns a short label describing where the artifact came from.
func (r ArtifactRecord) SourceLabel() string {
	switch {
	case r.Declared && r.Discovered:
		return "declared, discovered"
	case r.Declared:
		return "declared"
	case r.Discovered:
		return "discovered"
	default:
		return "unknown"
	}
}

// StateLabel returns the current filesystem state of the artifact.
func (r ArtifactRecord) StateLabel() string {
	switch {
	case r.Problem != "":
		return "problem"
	case r.Present:
		return "present"
	default:
		return "missing"
	}
}

// SizeLabel formats the artifact size for compact UI display.
func (r ArtifactRecord) SizeLabel() string {
	if !r.Present || r.SizeBytes <= 0 {
		return "-"
	}
	return FormatArtifactSize(r.SizeBytes)
}

// PathLabel returns the preferred path label for UI rendering.
func (r ArtifactRecord) PathLabel() string {
	return r.Path.Display()
}

// ArtifactSummary aggregates the artifact records for a step execution.
type ArtifactSummary struct {
	BaseDir   string
	StepID    string
	StepTitle string
	Records   []ArtifactRecord
}

// Count returns the total number of artifact records.
func (s ArtifactSummary) Count() int {
	return len(s.Records)
}

// PresentCount returns how many records point to files that exist.
func (s ArtifactSummary) PresentCount() int {
	count := 0
	for _, record := range s.Records {
		if record.Present {
			count++
		}
	}
	return count
}

// MissingCount returns how many records are absent on disk.
func (s ArtifactSummary) MissingCount() int {
	count := 0
	for _, record := range s.Records {
		if !record.Present && record.Problem == "" {
			count++
		}
	}
	return count
}

// ProblemCount returns how many records could not be inspected cleanly.
func (s ArtifactSummary) ProblemCount() int {
	count := 0
	for _, record := range s.Records {
		if record.Problem != "" {
			count++
		}
	}
	return count
}

// TotalBytes returns the sum of all present artifact sizes.
func (s ArtifactSummary) TotalBytes() int64 {
	var total int64
	for _, record := range s.Records {
		if record.Present {
			total += record.SizeBytes
		}
	}
	return total
}

// Brief renders a compact description that is useful in the UI footer or
// summary cards.
func (s ArtifactSummary) Brief() string {
	parts := []string{fmt.Sprintf("%d artifact(s)", s.Count())}
	if present := s.PresentCount(); present > 0 {
		parts = append(parts, fmt.Sprintf("%d present", present))
	}
	if missing := s.MissingCount(); missing > 0 {
		parts = append(parts, fmt.Sprintf("%d missing", missing))
	}
	if problems := s.ProblemCount(); problems > 0 {
		parts = append(parts, fmt.Sprintf("%d problem(s)", problems))
	}
	if total := s.TotalBytes(); total > 0 {
		parts = append(parts, FormatArtifactSize(total))
	}
	return strings.Join(parts, ", ")
}

// TableRows returns ready-to-render rows for a future table widget.
func (s ArtifactSummary) TableRows() [][]string {
	rows := make([][]string, 0, len(s.Records))
	for _, record := range s.sortedRecords() {
		rows = append(rows, []string{
			record.Name,
			record.PathLabel(),
			record.SourceLabel(),
			record.StateLabel(),
			record.SizeLabel(),
		})
	}
	return rows
}

func (s ArtifactSummary) sortedRecords() []ArtifactRecord {
	records := append([]ArtifactRecord(nil), s.Records...)
	sortArtifactRecords(records)
	return records
}

// NormalizeArtifactPath resolves a configured artifact path against the base
// directory and preserves a workspace-relative label when possible.
func NormalizeArtifactPath(baseDir string, rawPath string) (ArtifactPath, error) {
	baseDir = strings.TrimSpace(baseDir)
	rawPath = strings.TrimSpace(rawPath)
	if baseDir == "" {
		return ArtifactPath{}, fmt.Errorf("base directory is required")
	}
	if rawPath == "" {
		return ArtifactPath{}, fmt.Errorf("artifact path is required")
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return ArtifactPath{}, fmt.Errorf("resolve base directory: %w", err)
	}

	resolved := rawPath
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(baseAbs, resolved)
	}
	absolute := filepath.Clean(resolved)

	path := ArtifactPath{
		Input:    rawPath,
		Absolute: absolute,
	}
	if rel, err := filepath.Rel(baseAbs, absolute); err == nil && rel != "" {
		if rel == "." || !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".." {
			path.Relative = filepath.Clean(rel)
		}
	}

	return path, nil
}

// NormalizeArtifactDeclarations resolves explicit artifact declarations into
// filesystem-aware records.
func NormalizeArtifactDeclarations(
	baseDir string,
	declarations []ArtifactDeclaration,
) ([]ArtifactRecord, error) {
	records := make([]ArtifactRecord, 0, len(declarations))
	seen := make(map[string]struct{}, len(declarations))

	for _, decl := range declarations {
		record, err := artifactRecordFromDeclaration(baseDir, decl)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[record.Path.Absolute]; ok {
			continue
		}
		seen[record.Path.Absolute] = struct{}{}
		records = append(records, record)
	}

	sortArtifactRecords(records)
	return records, nil
}

// DiscoverArtifactRecords normalizes file paths that were found after a step
// finished execution.
func DiscoverArtifactRecords(baseDir string, discovered []string) ([]ArtifactRecord, error) {
	records := make([]ArtifactRecord, 0, len(discovered))
	seen := make(map[string]struct{}, len(discovered))

	for _, rawPath := range discovered {
		record, err := artifactRecordFromDiscovery(baseDir, rawPath)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[record.Path.Absolute]; ok {
			continue
		}
		seen[record.Path.Absolute] = struct{}{}
		records = append(records, record)
	}

	sortArtifactRecords(records)
	return records, nil
}

// BuildArtifactSummary merges explicit declarations and discovered files into a
// single artifact catalog for a step execution.
func BuildArtifactSummary(
	baseDir string,
	stepID string,
	stepTitle string,
	declarations []ArtifactDeclaration,
	discovered []string,
) (ArtifactSummary, error) {
	declaredRecords, err := NormalizeArtifactDeclarations(baseDir, declarations)
	if err != nil {
		return ArtifactSummary{}, err
	}
	discoveredRecords, err := DiscoverArtifactRecords(baseDir, discovered)
	if err != nil {
		return ArtifactSummary{}, err
	}

	merged := make([]ArtifactRecord, 0, len(declaredRecords)+len(discoveredRecords))
	index := make(map[string]int, len(declaredRecords)+len(discoveredRecords))

	appendMerged := func(record ArtifactRecord) {
		key := record.Path.Absolute
		if key == "" {
			key = record.Path.Input
		}
		if i, ok := index[key]; ok {
			merged[i] = mergeArtifactRecords(merged[i], record)
			return
		}
		index[key] = len(merged)
		merged = append(merged, record)
	}

	for _, record := range declaredRecords {
		appendMerged(record)
	}
	for _, record := range discoveredRecords {
		appendMerged(record)
	}

	sortArtifactRecords(merged)

	return ArtifactSummary{
		BaseDir:   baseDir,
		StepID:    stepID,
		StepTitle: stepTitle,
		Records:   merged,
	}, nil
}

func artifactRecordFromDeclaration(
	baseDir string,
	decl ArtifactDeclaration,
) (ArtifactRecord, error) {
	path, err := NormalizeArtifactPath(baseDir, decl.Path)
	if err != nil {
		return ArtifactRecord{}, err
	}

	record := ArtifactRecord{
		Name:        strings.TrimSpace(decl.Name),
		Description: strings.TrimSpace(decl.Description),
		Path:        path,
		Declared:    true,
		Optional:    decl.Optional,
	}
	if record.Name == "" {
		record.Name = artifactName(path)
	}

	populateArtifactStat(&record)
	return record, nil
}

func artifactRecordFromDiscovery(baseDir string, rawPath string) (ArtifactRecord, error) {
	path, err := NormalizeArtifactPath(baseDir, rawPath)
	if err != nil {
		return ArtifactRecord{}, err
	}

	record := ArtifactRecord{
		Name:       artifactName(path),
		Path:       path,
		Discovered: true,
	}
	populateArtifactStat(&record)
	return record, nil
}

func populateArtifactStat(record *ArtifactRecord) {
	info, err := os.Stat(record.Path.Absolute)
	if err == nil {
		record.Present = true
		record.SizeBytes = info.Size()
		record.ModTime = info.ModTime()
		return
	}
	if !os.IsNotExist(err) {
		record.Problem = err.Error()
	}
}

func mergeArtifactRecords(existing, incoming ArtifactRecord) ArtifactRecord {
	if existing.Name == "" {
		existing.Name = incoming.Name
	}
	if existing.Description == "" {
		existing.Description = incoming.Description
	}
	if existing.Path.Input == "" {
		existing.Path = incoming.Path
	}
	existing.Declared = existing.Declared || incoming.Declared
	existing.Discovered = existing.Discovered || incoming.Discovered
	existing.Optional = existing.Optional || incoming.Optional
	existing.Present = existing.Present || incoming.Present
	if existing.SizeBytes == 0 {
		existing.SizeBytes = incoming.SizeBytes
	}
	if existing.ModTime.IsZero() {
		existing.ModTime = incoming.ModTime
	}
	if existing.Problem == "" {
		existing.Problem = incoming.Problem
	}
	return existing
}

func sortArtifactRecords(records []ArtifactRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		left := records[i]
		right := records[j]
		if left.Path.Display() == right.Path.Display() {
			if left.Name == right.Name {
				return left.SourceLabel() < right.SourceLabel()
			}
			return left.Name < right.Name
		}
		return left.Path.Display() < right.Path.Display()
	})
}

func artifactName(path ArtifactPath) string {
	if path.Relative != "" && path.Relative != "." {
		return filepath.Base(path.Relative)
	}
	if path.Absolute != "" {
		return filepath.Base(path.Absolute)
	}
	return path.Input
}

// FormatArtifactSize renders a size using binary units and keeps bytes compact
// for future tabular display.
func FormatArtifactSize(sizeBytes int64) string {
	if sizeBytes < 0 {
		sizeBytes = 0
	}

	const unit = int64(1024)
	if sizeBytes < unit {
		return fmt.Sprintf("%d B", sizeBytes)
	}

	sizes := []struct {
		label string
		div   int64
	}{
		{label: "KiB", div: unit},
		{label: "MiB", div: unit * unit},
		{label: "GiB", div: unit * unit * unit},
	}
	for i := len(sizes) - 1; i >= 0; i-- {
		if sizeBytes >= sizes[i].div {
			value := float64(sizeBytes) / float64(sizes[i].div)
			return fmt.Sprintf("%.1f %s", value, sizes[i].label)
		}
	}

	return fmt.Sprintf("%d B", sizeBytes)
}
