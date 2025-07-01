package profile

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/spf13/cobra"

	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// NewProfileCommand creates the profile command for performance analysis
func NewProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "profile",
		Aliases: []string{"p"},
		Short:   "Performance profiling and benchmarking tools",
		Long:    `Tools for analyzing performance and memory usage of mdnotes operations`,
		Hidden:  true, // Hidden utility command
	}

	cmd.AddCommand(newCPUProfileCommand())
	cmd.AddCommand(newMemoryProfileCommand())
	cmd.AddCommand(newBenchmarkCommand())

	return cmd
}

func newCPUProfileCommand() *cobra.Command {
	var (
		outputFile string
		duration   time.Duration
	)

	cmd := &cobra.Command{
		Use:   "cpu [vault-path]",
		Short: "Generate CPU profile during operation",
		Long:  `Run a sample operation while generating a CPU profile for performance analysis`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Create CPU profile
			f, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("creating profile file: %w", err)
			}
			defer f.Close()

			if err := pprof.StartCPUProfile(f); err != nil {
				return fmt.Errorf("starting CPU profile: %w", err)
			}
			defer pprof.StopCPUProfile()

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Scan vault
			scanner := vault.NewScanner(vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns))
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			fmt.Printf("Profiling CPU usage for %d files...\n", len(files))

			// Run sample operations
			start := time.Now()
			err = runSampleOperations(files)
			elapsed := time.Since(start)

			if err != nil {
				return fmt.Errorf("running operations: %w", err)
			}

			fmt.Printf("Completed in %v\n", elapsed)
			fmt.Printf("CPU profile written to %s\n", outputFile)
			fmt.Printf("Analyze with: go tool pprof %s\n", outputFile)

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "cpu.prof", "Output file for CPU profile")
	cmd.Flags().DurationVarP(&duration, "duration", "d", 30*time.Second, "Duration to run profile")

	return cmd
}

func newMemoryProfileCommand() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "memory [vault-path]",
		Short: "Generate memory profile during operation",
		Long:  `Run a sample operation while generating a memory profile for analysis`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Scan vault
			scanner := vault.NewScanner(vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns))
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			fmt.Printf("Profiling memory usage for %d files...\n", len(files))

			// Run sample operations
			err = runSampleOperations(files)
			if err != nil {
				return fmt.Errorf("running operations: %w", err)
			}

			// Force garbage collection and create memory profile
			runtime.GC()

			f, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("creating profile file: %w", err)
			}
			defer f.Close()

			if err := pprof.WriteHeapProfile(f); err != nil {
				return fmt.Errorf("writing memory profile: %w", err)
			}

			fmt.Printf("Memory profile written to %s\n", outputFile)
			fmt.Printf("Analyze with: go tool pprof %s\n", outputFile)

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "memory.prof", "Output file for memory profile")

	return cmd
}

func newBenchmarkCommand() *cobra.Command {
	var (
		iterations     int
		fileCount      int
		workers        int
		enableParallel bool
	)

	cmd := &cobra.Command{
		Use:   "benchmark [vault-path]",
		Short: "Run performance benchmarks",
		Long:  `Execute performance benchmarks on various operations`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Generate test vault if specified
			var files []*vault.VaultFile

			if fileCount > 0 {
				fmt.Printf("Generating test vault with %d files...\n", fileCount)
				files = generateTestVault(fileCount)
			} else {
				// Load existing vault
				cfg, err := loadConfig(cmd)
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}

				scanner := vault.NewScanner(vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns))
				files, err = scanner.Walk(vaultPath)
				if err != nil {
					return fmt.Errorf("scanning vault: %w", err)
				}
			}

			fmt.Printf("Running benchmarks on %d files with %d iterations...\n", len(files), iterations)

			// Run benchmarks
			results := runBenchmarks(files, iterations, workers, enableParallel)

			// Display results
			displayBenchmarkResults(results)

			return nil
		},
	}

	cmd.Flags().IntVarP(&iterations, "iterations", "i", 5, "Number of benchmark iterations")
	cmd.Flags().IntVarP(&fileCount, "generate", "g", 0, "Generate test vault with N files")
	cmd.Flags().IntVarP(&workers, "workers", "w", runtime.NumCPU(), "Number of workers for parallel tests")
	cmd.Flags().BoolVarP(&enableParallel, "parallel", "p", true, "Enable parallel processing tests")

	return cmd
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")

	if configPath != "" {
		return config.LoadConfigFromFile(configPath)
	}

	return config.LoadConfigWithFallback(config.GetDefaultConfigPaths())
}

func runSampleOperations(files []*vault.VaultFile) error {
	// Note: not using vault variable for now, processing files directly

	// Run frontmatter operations
	frontmatterProcessor := processor.NewFrontmatterProcessor()
	for _, file := range files {
		frontmatterProcessor.Ensure(file, "tags", []string{})
	}

	// Run type casting
	caster := processor.NewTypeCaster()
	for _, file := range files {
		for _, value := range file.Frontmatter {
			if strVal, ok := value.(string); ok {
				detectedType := caster.AutoDetect(strVal)
				if detectedType != "string" {
					caster.Cast(strVal, detectedType)
				}
			}
		}
	}

	// Run heading analysis
	headingProcessor := processor.NewHeadingProcessor()
	for _, file := range files {
		headingProcessor.Analyze(file)
	}

	// Run link parsing
	linkParser := processor.NewLinkParser()
	for _, file := range files {
		linkParser.Extract(file.Body)
	}

	return nil
}

func generateTestVault(fileCount int) []*vault.VaultFile {
	files := make([]*vault.VaultFile, fileCount)

	for i := 0; i < fileCount; i++ {
		files[i] = &vault.VaultFile{
			Path:         fmt.Sprintf("test/file_%d.md", i),
			RelativePath: fmt.Sprintf("file_%d.md", i),
			Modified:     time.Now(),
			Frontmatter: map[string]interface{}{
				"title": fmt.Sprintf("Test File %d", i),
				"id":    fmt.Sprintf("test-%d", i),
				"tags":  []string{"test", "benchmark"},
			},
			Body: fmt.Sprintf(`# Test File %d

This is a test file for benchmarking purposes.

## Section 1

Content with [[link_%d]] and [markdown link](file_%d.md).

## Section 2

More content with ![[image_%d.png]] and additional text.

Some text with **bold** and *italic* formatting.

- List item 1
- List item 2  
- List item 3

### Subsection

Final content section.
`, i, (i+1)%fileCount, (i+1)%fileCount, i),
		}

		// Parse content
		content := fmt.Sprintf(`---
title: Test File %d
id: test-%d
tags: [test, benchmark]
---
%s`, i, i, files[i].Body)

		files[i].Parse([]byte(content))
	}

	return files
}

type BenchmarkResult struct {
	Name     string
	Duration time.Duration
	Files    int
	Parallel bool
	Workers  int
}

func runBenchmarks(files []*vault.VaultFile, iterations, workers int, enableParallel bool) []BenchmarkResult {
	var results []BenchmarkResult

	vault := &processor.Vault{
		Files: files,
		Path:  "/test/vault",
	}

	// Frontmatter Ensure Benchmark
	results = append(results, benchmarkOperation("Frontmatter.Ensure", func() error {
		processor := processor.NewFrontmatterProcessor()
		for _, file := range files {
			processor.Ensure(file, "tags", []string{})
		}
		return nil
	}, iterations, len(files), false, 1))

	// Type Casting Benchmark
	results = append(results, benchmarkOperation("Type.Cast", func() error {
		caster := processor.NewTypeCaster()
		for _, file := range files {
			for _, value := range file.Frontmatter {
				if strVal, ok := value.(string); ok {
					caster.AutoDetect(strVal)
				}
			}
		}
		return nil
	}, iterations, len(files), false, 1))

	// Heading Analysis Benchmark
	results = append(results, benchmarkOperation("Heading.Analyze", func() error {
		processor := processor.NewHeadingProcessor()
		for _, file := range files {
			processor.Analyze(file)
		}
		return nil
	}, iterations, len(files), false, 1))

	// Link Parsing Benchmark
	results = append(results, benchmarkOperation("Link.Parse", func() error {
		parser := processor.NewLinkParser()
		for _, file := range files {
			parser.Extract(file.Body)
		}
		return nil
	}, iterations, len(files), false, 1))

	// Parallel benchmarks if enabled
	if enableParallel && len(files) >= 50 {
		results = append(results, benchmarkParallelOperation("Frontmatter.Ensure.Parallel", vault, iterations, workers))
	}

	return results
}

func benchmarkOperation(name string, operation func() error, iterations, fileCount int, parallel bool, workers int) BenchmarkResult {
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		operation()
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)

	return BenchmarkResult{
		Name:     name,
		Duration: avgDuration,
		Files:    fileCount,
		Parallel: parallel,
		Workers:  workers,
	}
}

func benchmarkParallelOperation(name string, vault *processor.Vault, iterations, workers int) BenchmarkResult {
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Create parallel processor
		baseProcessor := &processor.FrontmatterEnsureProcessor{}
		parallelProcessor := processor.NewParallelProcessor(baseProcessor, workers)

		params := map[string]interface{}{
			"field":   "tags",
			"default": []string{},
		}

		parallelProcessor.Process(nil, vault, params)
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)

	return BenchmarkResult{
		Name:     name,
		Duration: avgDuration,
		Files:    len(vault.Files),
		Parallel: true,
		Workers:  workers,
	}
}

func displayBenchmarkResults(results []BenchmarkResult) {
	fmt.Printf("\n%-30s %12s %8s %10s %8s\n", "Benchmark", "Duration", "Files", "Parallel", "Workers")
	fmt.Printf("%-30s %12s %8s %10s %8s\n", "─────────", "────────", "─────", "────────", "───────")

	for _, result := range results {
		parallel := "No"
		if result.Parallel {
			parallel = "Yes"
		}

		fmt.Printf("%-30s %12v %8d %10s %8d\n",
			result.Name,
			result.Duration,
			result.Files,
			parallel,
			result.Workers,
		)
	}

	// Calculate files per second for the largest benchmark
	if len(results) > 0 {
		fastest := results[0]
		for _, result := range results {
			if result.Duration < fastest.Duration {
				fastest = result
			}
		}

		filesPerSecond := float64(fastest.Files) / fastest.Duration.Seconds()
		fmt.Printf("\nFastest operation: %s\n", fastest.Name)
		fmt.Printf("Throughput: %.0f files/second\n", filesPerSecond)
	}
}
