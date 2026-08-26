package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"rayo/golden"
	"rayo/internal/compile"
	"rayo/repl"
	"rayo/tools/lsp"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"

	includePaths []string
	outputDir    string
	verbose      bool
	emitGo       bool
	testDir      string
	testFilter   string
)

func transpileFile(inputFile string) error {
	opts := compile.Options{IncludePaths: includePaths}
	compiled, err := compile.BuildProgram(inputFile, opts)
	if err != nil {
		return err
	}

	outputFile := outputDir
	if outputFile == "" {
		outputFile = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ".go"
	}

	if err := os.WriteFile(outputFile, []byte(compiled), 0o644); err != nil {
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
	opts := compile.Options{IncludePaths: includePaths}
	compiled, err := compile.BuildProgram(inputFile, opts)
	if err != nil {
		return err
	}

	tempGoFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_temp.go"
	if err := os.WriteFile(tempGoFile, []byte(compiled), 0o644); err != nil {
		return fmt.Errorf("failed to write temp file %s: %w", tempGoFile, err)
	}
	defer os.Remove(tempGoFile)

	if verbose {
		fmt.Printf("Generated temporary file: %s\n", tempGoFile)
		fmt.Printf("Running Go code...\n")
	}

	cmd := exec.Command("go", append([]string{"run", tempGoFile}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGoldenTests() error {
	dir := testDir
	if dir == "" {
		dir = "testdata/golden"
	}
	if err := golden.Run(dir, testFilter); err != nil {
		return err
	}
	fmt.Println("golden: ok")
	return nil
}

func main() {
	var rootCmd = &cobra.Command{
		Use:     "rayo",
		Short:   "Rayo Transpiler CLI",
		Version: version,
	}

	rootCmd.PersistentFlags().StringSliceVarP(&includePaths, "include", "I", nil, "Include paths for resolving .ryo imports")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "", "Output directory or file for transpile")
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

	testCmd := &cobra.Command{
		Use:   "test [flags] [case-substring]",
		Short: "Run golden tests (same checks as go test -run TestGolden ./golden)",
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 && testFilter == "" {
				testFilter = args[0]
			}
			if err := runGoldenTests(); err != nil {
				fmt.Fprintf(os.Stderr, "golden: %v\n", err)
				os.Exit(1)
			}
		},
	}
	testCmd.Flags().StringVar(&testDir, "dir", "", "Golden directory (default: testdata/golden)")
	testCmd.Flags().StringVar(&testFilter, "case", "", "Substring filter on golden case name")
	rootCmd.AddCommand(testCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "repl",
		Short: "Start the interactive Rayo REPL",
		Run: func(cmd *cobra.Command, args []string) {
			r := repl.NewREPL(compile.Options{IncludePaths: includePaths})
			r.Run()
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("rayo %s\n", version)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Built: %s\n", date)
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "lsp [address]",
		Short: "Run the Language Server Protocol server",
		Long:  "Starts the LSP server for .ryo files. Default address is :2087 (used by the Rayo VS Code client).",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			addr := ":2087"
			if len(args) > 0 {
				addr = args[0]
			}
			if err := lsp.RunServer(addr); err != nil {
				fmt.Fprintf(os.Stderr, "LSP server: %v\n", err)
				os.Exit(1)
			}
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
