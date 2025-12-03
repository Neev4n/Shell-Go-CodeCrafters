package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codecrafters-io/shell-starter-go/internal/shell"
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var _ = os.Stdout

func main() {

	s := shell.New(os.Stdin, os.Stdout, os.Stderr)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}

}
