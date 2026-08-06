// zsh_poundai reads the current shell editing buffer on stdin and a cursor offset as
// its argument, asks the configured LLM to complete the command, and prints
// the text to insert at the cursor.
package main

import (
	"fmt"
	"os"

	"github.com/elee1766/zsh_poundai/pkg/cli"
)

func main() {
	if err := cli.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "zsh_poundai: %v\n", err)
		os.Exit(1)
	}
}
