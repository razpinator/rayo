package io

import (
    "os"
    "io/ioutil"
)

func ReadText(path string) (string, error) {
    b, err := ioutil.ReadFile(path)
    return string(b), err
}

func WriteText(path, data string) error {
    return ioutil.WriteFile(path, []byte(data), 0644)
}

func ReadBytes(path string) ([]byte, error) {
    return ioutil.ReadFile(path)
}

func WriteBytes(path string, data []byte) error {
    return ioutil.WriteFile(path, data, 0644)
}
