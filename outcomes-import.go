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

type request struct {
	body     string
	apikey   string
	domain   string
	method   string
	endpoint string
}

type importable_guid struct {
	Title string `json:"title"`
	Guid  string `json:"guid"`
}

type migration_issue struct {
	Id               int    `json:"id"`
	Issue_type       string `json:"issue_type"`
	Description      string `json:"description"`
	Error_report_url string `json:"error_report_html_url"`
	Error_message    string `json:"error_message"`
}

type migration_status struct {
	Id                     int               `json:"id"`
	Workflow_state         string            `json:"workflow_state"`
	Migration_issues_count int               `json:"migration_issues_count"`
	Migration_issues       []migration_issue `json:"migration_issues"`
}

type new_import struct {
	Migration_id int    `json:"migration_id"`
	Guid         string `json:"guid"`
}

func main() {
	var apikey = flag.String("apikey", "", "Canvas API key")
	var domain = flag.String(
		"domain",
		"",
		"The domain.  You can just say the school name if they have a vanity domain, like 'utah' for 'utah.instructure.com'",
	)
	var status = flag.Int("status", 0, "migration ID to check status for")
	var available = flag.Bool("available", false, "Check available migration IDs")
	var guid = flag.String("guid", "", "GUID to schedule for import")
	flag.Parse()

	req := request{apikey: *apikey, domain: *domain}
	verify_request(&req)
	req.domain = normalize_domain(req.domain)

	if *available {
		get_available(req)
	} else if *status != 0 {
		get_status(req, *status)
	} else if *guid != "" {
		import_guid(req, *guid)
	} else {
		log.Fatalln("You didn't say whether you wanted to check available, a migration status, or schedule an import.  Not sure what to do ¯\\_(ツ)_/¯")
	}
}

func normalize_domain(domain string) string {
	retval := domain
	// if we start with http then don't add it, otherwise do
	if !strings.HasPrefix(retval, "http") {
		retval = fmt.Sprintf("https://%s", retval)
		if !strings.HasSuffix(retval, "com") && !strings.HasSuffix(retval, "/") {
			retval = fmt.Sprintf("%s.instructure.com", retval)
		}
	}
	return strings.TrimSuffix(retval, "/")
}

func err_and_Exit(message string) {
	flag.Usage()
	log.Fatalln(message)
	os.Exit(1)
}

func verify_request(req *request) {
	if req.apikey == "" {
		err_and_Exit("You need a valid canvas API key")
	}
	if req.domain == "" {
		err_and_Exit("You must supply a canvas domain")
	}
}

func http_request(req request) (*http.Client, *http.Request) {
	client := &http.Client{}
	hreq, err := http.NewRequest(
		req.method,
		fmt.Sprintf("%s%s", req.domain, req.endpoint),
		strings.NewReader(req.body),
	)
	if err != nil {
		log.Fatalln(err)
	}
	hreq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", req.apikey))
	return client, hreq
}

func get_available(req request) {
	req.body = ""
	req.method = "GET"
	req.endpoint = "/api/v1/global/outcomes_import/available"

	client, hreq := http_request(req)
	log.Printf("Requesting available guids from %s", hreq.URL)
	resp, err := client.Do(hreq)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	var guids []importable_guid
	if e := json.NewDecoder(resp.Body).Decode(&guids); e != nil {
		log.Fatalln(e)
	}
	print_importable_guids(guids)
}

func get_status(req request, migration_id int) {
	req.body = ""
	req.method = "GET"
	req.endpoint = fmt.Sprintf(
		"/api/v1/global/outcomes_import/migration_status/%d",
		migration_id,
	)

	client, hreq := http_request(req)

	log.Printf("Retrieving status for migration %d", migration_id)
	resp, err := client.Do(hreq)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	var mstatus migration_status
	if e := json.NewDecoder(resp.Body).Decode(&mstatus); e != nil {
		log.Fatalln(e)
	}
	print_migration_status(mstatus)
}

func import_guid(req request, guid string) {
	req.body = fmt.Sprintf("guid=%s", guid)
	req.method = "POST"
	req.endpoint = "/api/v1/global/outcomes_import/"

	client, hreq := http_request(req)

	log.Printf("Requesting import of GUID %s", guid)
	resp, err := client.Do(hreq)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	var nimport new_import
	if e := json.NewDecoder(resp.Body).Decode(&nimport); e != nil {
		log.Fatalln(e)
	}
	print_import_results(nimport)
}

func print_importable_guids(guids []importable_guid) {
	fmt.Printf("GUIDs available to import:\n\n")
	for _, guid := range guids {
		fmt.Printf("%s - %s\n", guid.Guid, guid.Title)
	}
}

func print_migration_status(mstatus migration_status) {
	fmt.Printf("\nMigration status for migration '%d':\n", mstatus.Id)
	fmt.Printf(" - Workflow state: %s\n", mstatus.Workflow_state)
	fmt.Printf(" - Migration issues count: %d\n", mstatus.Migration_issues_count)
	fmt.Printf(" - Migration issues:\n")
	for _, val := range mstatus.Migration_issues {
		fmt.Printf("   - ID: %d\n", val.Id)
		fmt.Printf("   - Link: %s\n", val.Error_report_url)
		fmt.Printf("   - Issue type: %s\n", val.Issue_type)
		fmt.Printf("   - Error message: %s\n", val.Error_message)
		fmt.Printf("   - Description: %s\n", val.Description)
	}
}

func print_import_results(nimport new_import) {
	fmt.Printf(
    "\nMigration ID is %d\n",
    nimport.Migration_id,
  )
}
