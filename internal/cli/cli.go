// Package cli builds the Rayo command-line interface. It is shared by both the
// `rayo` and `rayoc` binaries so they expose an identical command set; the two
// entry points differ only in the program name reported in help/usage.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"rayo/golden"
	"rayo/internal/compile"
	"rayo/internal/diag"
	"rayo/internal/parse"
	"rayo/repl"
	rfmt "rayo/tools/fmt"
	"rayo/tools/lint"
	"rayo/tools/lsp"

	"github.com/spf13/cobra"
)

// options holds the flags shared across commands for a single CLI invocation.
type options struct {
	includePaths []string
	outputDir    string
	verbose      bool
	emitGo       bool
	testDir      string
	testFilter   string
}

func (o *options) transpileFile(inputFile string) error {
	compiled, err := compile.BuildProgram(inputFile, compile.Options{IncludePaths: o.includePaths})
	if err != nil {
		return err
	}
	outputFile := o.outputDir
	if outputFile == "" {
		outputFile = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ".go"
	}
	if err := os.WriteFile(outputFile, []byte(compiled), 0o644); err != nil {
		return fmt.Errorf("failed to write output file %s: %w", outputFile, err)
	}
	if o.verbose {
		fmt.Printf("Transpiled %s -> %s\n", inputFile, outputFile)
	} else {
		fmt.Printf("Generated %s\n", outputFile)
	}
	return nil
}

func (o *options) runFile(inputFile string, args []string) error {
	compiled, err := compile.BuildProgram(inputFile, compile.Options{IncludePaths: o.includePaths})
	if err != nil {
		return err
	}
	tempGoFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_temp.go"
	if err := os.WriteFile(tempGoFile, []byte(compiled), 0o644); err != nil {
		return fmt.Errorf("failed to write temp file %s: %w", tempGoFile, err)
	}
	defer os.Remove(tempGoFile)
	if o.verbose {
		fmt.Printf("Generated temporary file: %s\n", tempGoFile)
		fmt.Printf("Running Go code...\n")
	}
	cmd := exec.Command("go", append([]string{"run", tempGoFile}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (o *options) runGoldenTests() error {
	dir := o.testDir
	if dir == "" {
		dir = "testdata/golden"
	}
	if err := golden.Run(dir, o.testFilter); err != nil {
		return err
	}
	fmt.Println("golden: ok")
	return nil
}

// collectRyoFiles expands the given paths into a list of .ryo files. Directory
// arguments are walked recursively; file arguments are taken as-is. With no
// paths, the current directory is walked.
func collectRyoFiles(paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	var files []string
	seen := map[string]bool{}
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			err := filepath.Walk(p, func(path string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !fi.IsDir() && strings.HasSuffix(path, ".ryo") {
					add(path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}
		add(p)
	}
	return files, nil
}

// fmtFiles formats the given files. In check mode it reports files that are not
// formatted and returns a non-nil error if any differ; otherwise it rewrites
// files in place (or prints to stdout when writeStdout is true).
func fmtFiles(files []string, check, writeStdout bool) error {
	var unformatted []string
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		out := rfmt.Format(string(src))
		switch {
		case writeStdout:
			fmt.Print(out)
		case check:
			if out != string(src) {
				unformatted = append(unformatted, f)
			}
		default:
			if out != string(src) {
				if err := os.WriteFile(f, []byte(out), 0o644); err != nil {
					return err
				}
				fmt.Printf("formatted %s\n", f)
			}
		}
	}
	if check && len(unformatted) > 0 {
		for _, f := range unformatted {
			fmt.Fprintf(os.Stderr, "not formatted: %s\n", f)
		}
		return fmt.Errorf("%d file(s) not formatted", len(unformatted))
	}
	return nil
}

// lintFiles lints the given files, printing findings. It returns a non-nil
// error when any error-severity finding is present (so CI can gate on it).
func lintFiles(files []string) error {
	total := 0
	hadError := false
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		p := parse.NewParser(string(src))
		mod := p.ParseModule()
		if len(p.Errors()) > 0 {
			fmt.Fprintf(os.Stderr, "%s: parse errors: %v\n", f, p.Errors())
		}
		if mod == nil {
			continue
		}
		mod.Name = f
		for _, r := range lint.LintModule(mod) {
			total++
			fmt.Printf("%s:%d: %s\n", f, r.Span.Start.Line, r.String())
			if r.Severity == diag.SeverityError {
				hadError = true
			}
		}
	}
	if total == 0 {
		fmt.Println("lint: no issues found")
	}
	if hadError {
		return fmt.Errorf("lint found error-level issues")
	}
	return nil
}

// BuildRootCmd constructs the full Rayo command tree. name is the program name
// shown in usage ("rayo" or "rayoc"); version/commit/date populate the version
// command and --version flag.
func BuildRootCmd(name, version, commit, date string) *cobra.Command {
	o := &options{}

	rootCmd := &cobra.Command{
		Use:     name,
		Short:   "Rayo Transpiler CLI",
		Version: version,
	}
	rootCmd.PersistentFlags().StringSliceVarP(&o.includePaths, "include", "I", nil, "Include paths for resolving .ryo imports")
	rootCmd.PersistentFlags().StringVarP(&o.outputDir, "output", "o", "", "Output directory or file for transpile")
	rootCmd.PersistentFlags().BoolVarP(&o.verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&o.emitGo, "emit-go", false, "Emit Go code")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "transpile [file]",
		Short: "Transpile to Go",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := o.transpileFile(args[0]); err != nil {
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
			if err := o.runFile(args[0], args[1:]); err != nil {
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
			if len(args) > 0 && o.testFilter == "" {
				o.testFilter = args[0]
			}
			if err := o.runGoldenTests(); err != nil {
				fmt.Fprintf(os.Stderr, "golden: %v\n", err)
				os.Exit(1)
			}
		},
	}
	testCmd.Flags().StringVar(&o.testDir, "dir", "", "Golden directory (default: testdata/golden)")
	testCmd.Flags().StringVar(&o.testFilter, "case", "", "Substring filter on golden case name")
	rootCmd.AddCommand(testCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "repl",
		Short: "Start the interactive Rayo REPL",
		Run: func(cmd *cobra.Command, args []string) {
			repl.NewREPL(compile.Options{IncludePaths: o.includePaths}).Run()
		},
	})

	var fmtCheck, fmtStdout bool
	fmtCmd := &cobra.Command{
		Use:   "fmt [paths...]",
		Short: "Format .ryo source files",
		Long: "Formats Rayo source files with the stable, idempotent formatter.\n\n" +
			"With no paths, formats every .ryo file under the current directory.\n" +
			"By default files are rewritten in place; use --check to verify formatting\n" +
			"(non-zero exit if any file differs) or --stdout to print without writing.",
		Run: func(cmd *cobra.Command, args []string) {
			files, err := collectRyoFiles(args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := fmtFiles(files, fmtCheck, fmtStdout); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	fmtCmd.Flags().BoolVar(&fmtCheck, "check", false, "Report unformatted files without rewriting (exit 1 if any)")
	fmtCmd.Flags().BoolVar(&fmtStdout, "stdout", false, "Write formatted output to stdout instead of the file")
	rootCmd.AddCommand(fmtCmd)

	lintCmd := &cobra.Command{
		Use:   "lint [paths...]",
		Short: "Lint .ryo source files",
		Long: "Runs Rayo's linter over source files, reporting findings by stable rule\n" +
			"ID (RY001..RY005). With no paths, lints every .ryo file under the current\n" +
			"directory. Exits non-zero when any error-severity finding is present.",
		Run: func(cmd *cobra.Command, args []string) {
			files, err := collectRyoFiles(args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := lintFiles(files); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	rootCmd.AddCommand(lintCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s %s\n", name, version)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Built: %s\n", date)
		},
	})

	var useStdio bool
	lspCmd := &cobra.Command{
		Use:   "lsp [address]",
		Short: "Run the Language Server Protocol server",
		Long: "Starts the LSP server for .ryo files.\n\n" +
			"By default it listens on TCP :2087 (used by the bundled Rayo VS Code client).\n" +
			"Pass --stdio to communicate over stdin/stdout instead, which suits editors\n" +
			"that spawn the server as a child process (Neovim, Emacs, Helix, etc.).",
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if useStdio {
				if err := lsp.RunServerStdio(); err != nil {
					fmt.Fprintf(os.Stderr, "LSP server (stdio): %v\n", err)
					os.Exit(1)
				}
				return
			}
			addr := ":2087"
			if len(args) > 0 {
				addr = args[0]
			}
			if err := lsp.RunServer(addr); err != nil {
				fmt.Fprintf(os.Stderr, "LSP server: %v\n", err)
				os.Exit(1)
			}
		},
	}
	lspCmd.Flags().BoolVar(&useStdio, "stdio", false, "Communicate over stdin/stdout instead of TCP")
	rootCmd.AddCommand(lspCmd)

	return rootCmd
}

// Execute builds and runs the CLI, exiting non-zero on error.
func Execute(name, version, commit, date string) {
	if err := BuildRootCmd(name, version, commit, date).Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
