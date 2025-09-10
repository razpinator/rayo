package main

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
)

var (
    includePaths []string
    outputDir string
    verbose bool
    emitGo bool
)

func main() {
    var rootCmd = &cobra.Command{
        Use:   "rayoc",
        Short: "Rayo Transpiler CLI",
    }

    rootCmd.PersistentFlags().StringSliceVarP(&includePaths, "include", "I", nil, "Include paths")
    rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "", "Output directory")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
    rootCmd.PersistentFlags().BoolVar(&emitGo, "emit-go", false, "Emit Go code")

    rootCmd.AddCommand(&cobra.Command{
        Use: "lex",
        Short: "Lex source file",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("Lexing not yet implemented.")
        },
    })
    rootCmd.AddCommand(&cobra.Command{
        Use: "parse",
        Short: "Parse source file",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("Parsing not yet implemented.")
        },
    })
    rootCmd.AddCommand(&cobra.Command{
        Use: "check",
        Short: "Check semantics",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("Semantic check not yet implemented.")
        },
    })
    rootCmd.AddCommand(&cobra.Command{
        Use: "transpile",
        Short: "Transpile to Go",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("Transpilation not yet implemented.")
        },
    })
    rootCmd.AddCommand(&cobra.Command{
        Use: "run",
        Short: "Transpile and run",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("Run not yet implemented.")
        },
    })
    rootCmd.AddCommand(&cobra.Command{
        Use: "test",
        Short: "Run golden tests",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("Test not yet implemented.")
        },
    })

    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
