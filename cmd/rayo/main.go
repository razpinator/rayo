package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"rayo/internal/gen"
	"rayo/internal/parse"

	"github.com/spf13/cobra"
)

var (
	// Version information - can be set at build time with -ldflags
	version = "dev"
	commit  = "unknown"
	date    = "unknown"

	// Command line flags
	includePaths []string
	outputDir    string
	verbose      bool
	emitGo       bool
)

func compileWithDependencies(inputFile string) (string, error) {
	visited := make(map[string]bool)
	var allFunctions []string
	var allImports []string

	err := collectModules(inputFile, visited, &allFunctions, &allImports)
	if err != nil {
		return "", err
	}

	// Build final Go code
	var result strings.Builder
	result.WriteString("package main\n\n")

	// Add unique imports
	importSet := make(map[string]bool)
	for _, imp := range allImports {
		if !importSet[imp] && !strings.HasSuffix(imp, ".ryo") {
			result.WriteString(fmt.Sprintf("import \"%s\"\n", imp))
			importSet[imp] = true
		}
	}

	// Add all functions
	for _, fn := range allFunctions {
		result.WriteString(fn)
		result.WriteString("\n")
	}

	return result.String(), nil
}

func collectModules(filename string, visited map[string]bool, functions *[]string, imports *[]string) error {
	if visited[filename] {
		return nil // Already processed
	}
	visited[filename] = true

	// Read the source file
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Parse the source
	parser := parse.NewParser(string(source))
	module := parser.ParseModule()

	if len(parser.Errors()) > 0 {
		return fmt.Errorf("parse errors in %s: %v", filename, parser.Errors())
	}

	// Check if this module uses print() which requires "fmt"
	for _, stmt := range module.Body {
		if gen.ContainsPrint(stmt) {
			*imports = append(*imports, "fmt")
			break
		}
	}

	// Process imports first
	for _, imp := range module.Imports {
		*imports = append(*imports, imp.Path)

		// If it's a local .ryo file, recursively process it
		if strings.HasSuffix(imp.Path, ".ryo") {
			importPath := imp.Path
			if strings.HasPrefix(importPath, "./") {
				// Make path relative to current file
				dir := filepath.Dir(filename)
				importPath = filepath.Join(dir, importPath[2:])
			}

			err := collectModules(importPath, visited, functions, imports)
			if err != nil {
				return err
			}
		}
	}

	// Generate functions from this module
	ctx := gen.NewGenContext("main")
	for _, stmt := range module.Body {
		var funcBuilder strings.Builder
		ctx.Code = &funcBuilder
		gen.EmitStmt(stmt, ctx)
		if funcBuilder.Len() > 0 {
			*functions = append(*functions, funcBuilder.String())
		}
	}

	return nil
}

func transpileFile(inputFile string) error {
	// Read and compile the main file and its dependencies
	compiledModules, err := compileWithDependencies(inputFile)
	if err != nil {
		return err
	}

	// Determine output file
	outputFile := outputDir
	if outputFile == "" {
		// Default: replace .ryo extension with .go
		outputFile = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ".go"
	}

	// Write the generated Go code
	err = ioutil.WriteFile(outputFile, []byte(compiledModules), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file %s: %w", outputFile, err)
	}

	if verbose {
		fmt.Printf("Transpiled %s -> %s\n", inputFile, outputFile)
	} else {
		fmt.Printf("Generated %s\n", outputFile)
	}

	return nil
}

func runFile(inputFile string, args []string) error {
	// Generate a temporary Go file
	tempGoFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_temp.go"

	// Compile with dependencies
	compiledModules, err := compileWithDependencies(inputFile)
	if err != nil {
		return err
	}

	// Write temporary Go file
	err = ioutil.WriteFile(tempGoFile, []byte(compiledModules), 0644)
	if err != nil {
		return fmt.Errorf("failed to write temp file %s: %w", tempGoFile, err)
	}

	// Clean up temp file when done
	defer os.Remove(tempGoFile)

	if verbose {
		fmt.Printf("Generated temporary file: %s\n", tempGoFile)
		fmt.Printf("Running Go code...\n")
	}

	// Run the Go code
	cmd := exec.Command("go", append([]string{"run", tempGoFile}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func main() {
	var rootCmd = &cobra.Command{
		Use:     "rayo",
		Short:   "Rayo Transpiler CLI",
		Version: version,
	}

	rootCmd.PersistentFlags().StringSliceVarP(&includePaths, "include", "I", nil, "Include paths")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "", "Output directory")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&emitGo, "emit-go", false, "Emit Go code")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "lex",
		Short: "Lex source file",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Lexing not yet implemented.")
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "parse",
		Short: "Parse source file",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Parsing not yet implemented.")
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Check semantics",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Semantic check not yet implemented.")
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "transpile [file]",
		Short: "Transpile to Go",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := transpileFile(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "run [file]",
		Short: "Transpile and run",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runFile(args[0], args[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "test",
		Short: "Run golden tests",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Test not yet implemented.")
		},
	})

	// Add version command with detailed information
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("rayo %s\n", version)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Built: %s\n", date)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
