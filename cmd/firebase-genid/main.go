package main

import (
	"fmt"
	"os"

	"github.com/knq/firebase"
)

func main() {
	fmt.Fprintf(os.Stdout, "%s\n", firebase.GeneratePushID())
}
