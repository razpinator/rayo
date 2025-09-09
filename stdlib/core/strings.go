package core

import "strings"

func StrLen(s string) int {
    return len(s)
}

func StrUpper(s string) string {
    return strings.ToUpper(s)
}

func StrLower(s string) string {
    return strings.ToLower(s)
}

func StrSplit(s, sep string) []string {
    return strings.Split(s, sep)
}
