package cli

import (
	"fmt"
	"os"
)

func runVersion(_ []string) int {
	fmt.Fprintf(os.Stdout, "pickle %s\n", Version)
	return exitOK
}
