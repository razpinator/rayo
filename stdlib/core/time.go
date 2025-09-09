package core

import "time"

func Now() time.Time {
    return time.Now()
}

func FormatTime(t time.Time, layout string) string {
    return t.Format(layout)
}

func ParseTime(layout, value string) (time.Time, error) {
    return time.Parse(layout, value)
}
