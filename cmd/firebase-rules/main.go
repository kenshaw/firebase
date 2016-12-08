package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/knq/firebase"
)

var (
	flagCredentials = flag.String("creds", "", "path to google service account credentials")
	flagRulesFile   = flag.String("rules", "rules.json", "path to rules file")
	flagNoSave      = flag.Bool("nosave", true, "don't save existing rules")
	flagClearRules  = flag.Bool("clear", false, "clear rules")
	flagClearValue  = flag.String("val", "false", "clear rule value")
)

func main() {
	var err error

	flag.Parse()

	// check credentials
	if *flagCredentials == "" {
		fmt.Fprintf(os.Stderr, "error: invalid credentials file\n")
		os.Exit(1)
	}

	// load rules
	buf := []byte(fmt.Sprintf(emptyRules, *flagClearValue, *flagClearValue))
	if !*flagClearRules {
		buf, err = ioutil.ReadFile(*flagRulesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	// create ref
	ref, err := firebase.NewDatabaseRef(
		firebase.GoogleServiceAccountCredentialsFile(*flagCredentials),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// save existing rules
	if *flagNoSave {
		existing, err := ref.GetRulesJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		err = ioutil.WriteFile(*flagRulesFile+"-old", existing, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	// set rules
	err = ref.SetRulesJSON(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// emptyRules are the empty rule set for firebase (allow no reads/writes).
const emptyRules = `{
  "rules": {
    ".read": "%s",
    ".write": "%s"
  }
}`
