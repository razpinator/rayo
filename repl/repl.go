package repl

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

type REPL struct {
    Scope map[string]any
    History []string
}

func NewREPL() *REPL {
    return &REPL{Scope: map[string]any{}, History: []string{}}
}

func (r *REPL) Run() {
    reader := bufio.NewReader(os.Stdin)
    fmt.Println("Functure REPL. Type :help for commands.")
    for {
        fmt.Print("> ")
        line, err := reader.ReadString('\n')
        if err != nil {
            break
        }
        line = strings.TrimSpace(line)
        r.History = append(r.History, line)
        if line == ":help" {
            fmt.Println(":type expr — show type\n:vars — show variables\n:quit — exit")
            continue
        }
        if line == ":quit" {
            break
        }
        if line == ":vars" {
            fmt.Println("Variables:")
            for k, v := range r.Scope {
                fmt.Printf("%s = %v\n", k, v)
            }
            continue
        }
        // TODO: parse, sem-check, transpile, run
        fmt.Println("(not yet implemented)")
    }
}
