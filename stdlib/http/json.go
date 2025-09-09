package http

import "encoding/json"

func JSON(status int, v any) (int, string) {
    b, _ := json.MarshalIndent(v, "", "  ")
    return status, string(b)
}
