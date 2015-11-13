package client

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func valid(path string) bool {
    return strings.HasSuffix(path, ".js") ||
        strings.HasSuffix(path, ".html") ||
        strings.HasSuffix(path, ".css")
}

func TestClient(t *testing.T) {
    filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
        if f.IsDir() {
            return nil
        }
        if !valid(path) {
            return nil
        }
        file, _ := os.Open(path)
        content, _ := ioutil.ReadAll(file)
        content2, err := GetContent("/" + path)
        fmt.Println("FILE: ", path)
        fmt.Println(string(content2[:10]))
        fmt.Println(string(content2[len(content2)-10:]))
        if err != nil {
            t.Error(err)
        }
        if bytes.Compare(content, []byte(content2)) != 0 {
            t.Error("file diff: ", f.Name())
        }
        return nil
    })
}
