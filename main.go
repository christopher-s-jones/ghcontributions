package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/christopher-s-jones/ghcontributions/reporting"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func main() {

	// Configure the command based on command line flags
	config, err := Configure()
	if err != nil {
		flag.Usage()
		log.Fatalf("Couldn't parse the command line arguments: %s\n", err)
	}

	// Load and set Github API tokens per user
	var jsonBytes []byte
	if config.credentialsAreEncrypted {
		jsonBytes, err = exec.Command("gpg", "-d", config.credentialsFilePath).Output()
		if err != nil {
			flag.Usage()
			log.Fatalf("Couldn't decrypt the credentials file: %s", err)
		}
	} else {
		jsonBytes, err = os.ReadFile(config.credentialsFilePath)
		if err != nil {
			flag.Usage()
			log.Fatalf("Couldn't read the credentials file: %s\n", err)
		}
	}

	// Build a Credentials object from the JSON file
	jsonStr := string(jsonBytes)
	credentials := &reporting.Credentials{}
	err = json.Unmarshal([]byte(jsonStr), credentials)
	if err != nil {
		log.Fatalf("Couldn't parse JSON credentials file: %s", err)
	}

	// List repositories for each user and get statistics
	var reporter reporting.Reporter
	for _, credential := range *credentials {
		src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: credential.Token})
		httpClient := oauth2.NewClient(context.Background(), src)
		apiClient := githubv4.NewClient(httpClient)
		firstYear := config.firstReportingYear
		lastYear := config.lastReportingYear
		reporter, err = reporter.NewReporter(apiClient, credential.Username, firstYear, lastYear)
		if err != nil {
			log.Fatalf("Couldn't create a reporter object: %s", err)
		}
		err = reporter.Collect()
		if err != nil {
			log.Print(err)
		}
	}
	aggregatedResults, _ := reporter.Report()
	log.Print(aggregatedResults)
}

// A simple configuration to store and pass command line settings
type Configuration struct {
	credentialsAreEncrypted bool
	credentialsFilePath     string
	firstReportingYear      int
	lastReportingYear       int
}

// Configure creates a simple configuration based on
// command line arguments, or uses defaults
func Configure() (config Configuration, err error) {

	config = Configuration{}

	// Handle incoming command line flags
	flag.BoolVar(&config.credentialsAreEncrypted,
		"encrypted",
		false,
		"Whether the credentials file is PGP encrypted.")

	flag.StringVar(&config.credentialsFilePath,
		"credentials",
		"gh-tokens.json",
		"The name of the file containing Github usernames \nand API token values")

	flag.IntVar(&config.firstReportingYear,
		"firstyear",
		2000,
		"The first year to summarize.")

	year := time.Now().Year()
	flag.IntVar(&config.lastReportingYear,
		"lastyear",
		year,
		"The last year to summarize")

	// Build a sample credentials list for usage display
	cred := reporting.Credential{}
	cred.Username = "your-github-username"
	cred.Token = "your-github-api-token"
	cred2 := reporting.Credential{}
	cred2.Username = "your-next-github-username"
	cred2.Token = "your-next-github-api-token"
	creds := make([]reporting.Credential, 0)
	creds = append(creds, cred, cred2)

	// Provide adequate usage instructions
	flag.Usage = func() {
		fmt.Println("Github Summary Contributions reporter")
		fmt.Println("\n\tAggregates contributions across a list of Github accounts,")
		fmt.Println("\tand across a range of years,")
		fmt.Println("\tincluding total commits, total count of public repositories")
		fmt.Println("\tcontributed to, and total other contributions, including")
		fmt.Println("\tpull requests, merges, and issues.")
		fmt.Println("\nUsage:")
		fmt.Printf(" %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Println("")
		for range 40 {
			fmt.Print("-")
		}
		fmt.Println("")
		fmt.Println("\n1. Create a JSON file with a list of credentials:")
		credsStr, _ := json.MarshalIndent(creds, "", "  ")
		fmt.Printf("\n%s\n", credsStr)
		fmt.Println("\n2. Optionally encrypt the file using PGP/GPG,")
		fmt.Println("   and use the -encrypted flag if it is encrypted.")
		fmt.Println("\n3. Optionally set the -firstyear and -lastyear flags with four digit years.")
		fmt.Println("\n3. Pass the path to the file as the argument to the -credentials flag.")
	}

	// Read the command line arguments
	flag.Parse()

	return config, nil
}
