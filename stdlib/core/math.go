package core

import "math"

func Abs(x float64) float64 {
    return math.Abs(x)
}

func Pow(x, y float64) float64 {
    return math.Pow(x, y)
}

func Max(x, y float64) float64 {
    if x > y {
        return x
    }
    return y
}

func Min(x, y float64) float64 {
    if x < y {
        return x
    }
    return y
}
