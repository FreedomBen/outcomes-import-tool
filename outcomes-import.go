package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type config struct {
	Apikey      string
	MigrationId int
}

type request struct {
	Body     string
	Apikey   string
	Domain   string
	Method   string
	Endpoint string
}

type importableGuid struct {
	Title string `json:"title"`
	Guid  string `json:"guid"`
}

type migrationIssue struct {
	Id             int    `json:"id"`
	IssueType      string `json:"issue_type"`
	Description    string `json:"description"`
	ErrorReportUrl string `json:"error_report_html_url"`
	ErrorMessage   string `json:"error_message"`
}

type migrationStatus struct {
	Id                     int               `json:"id"`
	WorkflowState         string            `json:"workflow_state"`
	MigrationIssuesCount int               `json:"migration_issues_count"`
	MigrationIssues       []migrationIssue `json:"migration_issues"`
}

type newImport struct {
	Migration_id int    `json:"migration_id"`
	Guid         string `json:"guid"`
}

func main() {
	var apikey = flag.String("apikey", "", "Canvas API key")
	var domain = flag.String(
		"domain",
		"",
		"The domain.  You can just say the school name if they have a vanity domain, like 'utah' for 'utah.instructure.com' or 'localhost'",
	)
	var status = flag.Int("status", 0, "migration ID to check status for")
	var available = flag.Bool("available", false, "Check available migration IDs")
	var guid = flag.String("guid", "", "GUID to schedule for import")
	flag.Parse()

	// if the api key and last migration id are stored, use those

	req := request{Apikey: *apikey, Domain: *domain}
	verifyRequest(&req)
	req.Domain = normalizeDomain(req.Domain)

	if *available {
		getAvailable(req)
	} else if *status != 0 {
		getStatus(req, *status)
	} else if *guid != "" {
		importGuid(req, *guid)
	} else {
		log.Fatalln("You didn't say whether you wanted to check available, a migration status, or schedule an import.  Not sure what to do ¯\\_(ツ)_/¯")
	}
}

func normalizeDomain(domain string) string {
	retval := domain
  if domain == "localhost" {
    return "http://localhost:3000"
	// if we start with http then don't add it, otherwise do
  } else if !strings.HasPrefix(retval, "http") {
		retval = fmt.Sprintf("https://%s", retval)
		if !strings.HasSuffix(retval, "com") && !strings.HasSuffix(retval, "/") {
			retval = fmt.Sprintf("%s.instructure.com", retval)
		}
	}
	return strings.TrimSuffix(retval, "/")
}

func errAndExit(message string) {
	flag.Usage()
	log.Fatalln(message)
	os.Exit(1)
}

func verifyRequest(req *request) {
	if req.Apikey == "" {
		errAndExit("You need a valid canvas API key")
	}
	if req.Domain == "" {
		errAndExit("You must supply a canvas domain")
	}
}

func httpRequest(req request) (*http.Client, *http.Request) {
	client := &http.Client{}
	hreq, err := http.NewRequest(
		req.Method,
		fmt.Sprintf("%s%s", req.Domain, req.Endpoint),
		strings.NewReader(req.Body),
	)
	if err != nil {
		log.Fatalln(err)
	}
	hreq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", req.Apikey))
	return client, hreq
}

func getAvailable(req request) {
	req.Body = ""
	req.Method = "GET"
	req.Endpoint = "/api/v1/global/outcomes_import/available"

	client, hreq := httpRequest(req)
	log.Printf("Requesting available guids from %s", hreq.URL)
	resp, err := client.Do(hreq)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	var guids []importableGuid
	if e := json.NewDecoder(resp.Body).Decode(&guids); e != nil {
		log.Fatalln(e)
	}
	printImportableGuids(guids)
}

func getStatus(req request, migration_id int) {
	req.Body = ""
	req.Method = "GET"
	req.Endpoint = fmt.Sprintf(
		"/api/v1/global/outcomes_import/migration_status/%d",
		migration_id,
	)

	client, hreq := httpRequest(req)

	log.Printf("Retrieving status for migration %d", migration_id)
	resp, err := client.Do(hreq)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	var mstatus migrationStatus
	if e := json.NewDecoder(resp.Body).Decode(&mstatus); e != nil {
		log.Fatalln(e)
	}
	printMigrationStatus(mstatus)
}

func importGuid(req request, guid string) {
	req.Body = fmt.Sprintf("guid=%s", guid)
	req.Method = "POST"
	req.Endpoint = "/api/v1/global/outcomes_import/"

	client, hreq := httpRequest(req)

	log.Printf("Requesting import of GUID %s", guid)
	resp, err := client.Do(hreq)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	var nimport newImport
	if e := json.NewDecoder(resp.Body).Decode(&nimport); e != nil {
		log.Fatalln(e)
	}
	printImportResults(nimport)
}

func printImportableGuids(guids []importableGuid) {
	fmt.Printf("GUIDs available to import:\n\n")
	for _, guid := range guids {
		fmt.Printf("%s - %s\n", guid.Guid, guid.Title)
	}
}

func printMigrationStatus(mstatus migrationStatus) {
	fmt.Printf("\nMigration status for migration '%d':\n", mstatus.Id)
	fmt.Printf(" - Workflow state: %s\n", mstatus.WorkflowState)
	fmt.Printf(" - Migration issues count: %d\n", mstatus.MigrationIssuesCount)
	fmt.Printf(" - Migration issues:\n")
	for _, val := range mstatus.MigrationIssues {
		fmt.Printf("   - ID: %d\n", val.Id)
		fmt.Printf("   - Link: %s\n", val.ErrorReportUrl)
		fmt.Printf("   - Issue type: %s\n", val.IssueType)
		fmt.Printf("   - Error message: %s\n", val.ErrorMessage)
		fmt.Printf("   - Description: %s\n", val.Description)
	}
}

func printImportResults(nimport newImport) {
	fmt.Printf(
		"\nMigration ID is %d\n",
		nimport.Migration_id,
	)
}
