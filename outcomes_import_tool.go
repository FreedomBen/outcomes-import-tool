package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const (
	Version    string = "0.0.1"
	ConfigFile string = ".outcomes-import-tool.json"
)

type config struct {
	Apikey      string           `json:"apikey"`
	MigrationId int              `json:"migration_id"`
	Domain      string           `json:"domain"`
	Guids       []importableGuid `json:"guids"`
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
	Id                   int              `json:"id"`
	WorkflowState        string           `json:"workflow_state"`
	MigrationIssuesCount int              `json:"migration_issues_count"`
	MigrationIssues      []migrationIssue `json:"migration_issues"`
	Errors               []apiError       `json:"errors"`
}

type newImport struct {
	MigrationId int        `json:"migration_id"`
	Guid        string     `json:"guid"`
	Errors      []apiError `json:"errors"`
	Error       string     `json:"errors"`
}

type apiErrors struct {
	Errors []apiError `json:"errors"`
}

type apiError struct {
	Message string `json:"message"`
}

func fatalExit(message ...interface{}) {
	errmessage := make([]interface{}, len(message)+1)
	errmessage[0] = "[-]"
	for i, m := range message {
		errmessage[i+1] = m
	}
	fmt.Fprintln(os.Stderr, errmessage...)
	os.Exit(1)
}

func configFromFile() *config {
	if f, err := os.Open(configFile()); err == nil {
		var cf config
		if err := json.NewDecoder(f).Decode(&cf); err != nil {
			fatalExit("Config file json error:", err)
		}
		return &cf
	} else {
		if match, _ := regexp.MatchString("no such file or directory", err.Error()); match {
			writeBlankConfigFile()
		}
		return nil
	}
}

func writeBlankConfigFile() {
	c := &config{}
	b, _ := json.MarshalIndent(*c, "", "  ")
	ioutil.WriteFile(configFile(), b, 0600)
}

func (c *config) writeToFile() {
	current := configFromFile()
	// we only want to store the API key if the user already stores it
	if current == nil || current.Apikey == "" {
		c.Apikey = ""
	}
	b, err := json.MarshalIndent(*c, "", "  ")
	if err != nil {
		fatalExit("Error writing to", configFile())
	}
	ioutil.WriteFile(configFile(), b, 0600)
}

func configFile() string {
	return fmt.Sprintf("%s/%s", os.Getenv("HOME"), ConfigFile)
}

func main() {
	var apikey = flag.String("apikey", "", "Canvas API key")
	var domain = flag.String(
		"domain",
		"",
		"The domain.  You can just say the school name if they have a \"<school>.instructure.com\" domain, or 'localhost'",
	)
	var status = flag.Int("status", 0, "migration ID to check status")
	var available = flag.Bool("available", false, "Check available migration IDs")
	var guid = flag.String("guid", "", "GUID to schedule for import")
	var help = flag.Bool("help", false, "Print the help menu and exit")
	var version = flag.Bool("version", false, "Print the version and exit")
	flag.Parse()

	if *version {
		fmt.Println("[+] Outcomes Import Tool Version: ", Version)
		os.Exit(0)
	}

	if *help {
		printHelp()
		os.Exit(0)
	}

	if cf := configFromFile(); cf != nil {
		if *apikey == "" {
			fmt.Println("[+] Using API key from config file")
			apikey = &cf.Apikey
		}
		if *status == 0 {
			fmt.Println("[+] Using migration ID from config file")
			status = &cf.MigrationId
		}
		if *domain == "" {
			fmt.Println("[+] Using domain from config file")
			domain = &cf.Domain
		}
	}

	req := request{Apikey: *apikey, Domain: *domain}
	verifyRequest(&req)
	req.Domain = normalizeDomain(req.Domain)

	if *available {
		printAvailable(req)
	} else if *guid != "" {
		importGuid(req, *guid)
	} else if *status != 0 {
		getStatus(req, *status)
	} else {
		fatalExit("No recent migration ID, and none specified to query status on")
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

func errAndExit(message ...interface{}) {
	flag.Usage()
	fatalExit(message...)
}

func verifyRequest(req *request) {
	if req.Apikey == "" {
		errAndExit(fmt.Sprintf("Whoops, no API key stored in config file \"%s\" and none passed as an arg", configFile()))
	}
	if req.Domain == "" {
		errAndExit(fmt.Sprintf("Whoops, no canvas domain stored in config file \"%s\" and none passed as an arg", configFile()))
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
		fatalExit(err)
	}
	hreq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", req.Apikey))
	return client, hreq
}

func printAvailable(req request) {
	guids := getAvailable(req)
	printImportableGuids(guids)
	migId := 0
	if cff := configFromFile(); cff != nil {
		migId = cff.MigrationId
	}
	(&config{
		Apikey:      req.Apikey,
		Domain:      req.Domain,
		MigrationId: migId,
		Guids:       guids,
	}).writeToFile()
}

func getAvailable(req request) []importableGuid {
	req.Body = ""
	req.Method = "GET"
	req.Endpoint = "/api/v1/global/outcomes_import/available"

	client, hreq := httpRequest(req)
	fmt.Printf("[+] Requesting available guids from %s", hreq.URL)
	resp, err := client.Do(hreq)
	if err != nil {
		fatalExit(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	var errs apiErrors
	if json.NewDecoder(bytes.NewReader(body)).Decode(&errs); len(errs.Errors) > 0 {
		printErrors(errs.Errors)
		os.Exit(1)
	}

	var guids []importableGuid
	if e := json.NewDecoder(bytes.NewReader(body)).Decode(&guids); e != nil {
		fatalExit("JSON decoding error.  Make sure your API key is correct and that you have permission to read global outcomes", e)
	}
	return guids
}

func getStatus(req request, migrationId int) {
	req.Body = ""
	req.Method = "GET"
	req.Endpoint = fmt.Sprintf(
		"/api/v1/global/outcomes_import/migration_status/%d",
		migrationId,
	)

	client, hreq := httpRequest(req)

	fmt.Printf("[+] Retrieving status for migration %d\n", migrationId)
	resp, err := client.Do(hreq)
	if err != nil {
		fatalExit(err)
	}
	defer resp.Body.Close()

	var mstatus migrationStatus
	if e := json.NewDecoder(resp.Body).Decode(&mstatus); e != nil {
		fatalExit("JSON decoding error.  Make sure your API key is correct and that you have permission to read global outcomes", e)
	}
	printMigrationStatus(mstatus)
	prevConfig := configFromFile()
	(&config{
		Apikey:      req.Apikey,
		Domain:      req.Domain,
		MigrationId: migrationId,
		Guids:       prevConfig.Guids,
	}).writeToFile()
}

func importGuid(req request, guid string) {
	// first check to see if what we've been passed is a proper GUID
	guid = strings.ToUpper(guid)
	match, _ := regexp.MatchString(
		"[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}",
		guid,
	)

	if !match {
		fmt.Println("[+] GUID is not valid.  Checking to see if it matches a valid title...")
		// then check to see if we've been given a title
		config := configFromFile()
		var guids []importableGuid
		if len(config.Guids) > 0 {
			fmt.Println("[+] Using cached guid from config file.  Run tool with --available option to force refresh of GUIDs")
			guids = config.Guids
		} else {
			fmt.Println("[+] Cache file does not contain guids.  Fetching guids from AB")
			guids = getAvailable(req)
		}
		found := false
		for _, val := range guids {
			if strings.ToUpper(val.Title) == guid {
				guid = val.Guid
				found = true
				break
			}
		}
		if !found {
			fatalExit(fmt.Sprintf("\"%s\" is not a valid AB GUID and it did not match any titles", guid))
		}
	}

	req.Body = fmt.Sprintf("guid=%s", guid)
	req.Method = "POST"
	req.Endpoint = "/api/v1/global/outcomes_import/"

	client, hreq := httpRequest(req)

	fmt.Printf("[+] Requesting import of GUID %s", guid)
	resp, err := client.Do(hreq)
	if err != nil {
		fatalExit(err)
	}
	defer resp.Body.Close()

	var nimport newImport
	if e := json.NewDecoder(resp.Body).Decode(&nimport); e != nil {
		fatalExit("JSON decoding error.  Make sure your API key is correct and that you have permission to read global outcomes.", e)
	}
	printImportResults(nimport)
	prevConfig := configFromFile()
	(&config{
		Apikey:      req.Apikey,
		Domain:      req.Domain,
		MigrationId: nimport.MigrationId,
		Guids:       prevConfig.Guids,
	}).writeToFile()
}

func printImportableGuids(guids []importableGuid) {
	fmt.Printf("GUIDs available to import:\n\n")
	for _, guid := range guids {
		fmt.Printf("%s - %s\n", guid.Guid, guid.Title)
	}
}

func printMigrationStatus(mstatus migrationStatus) {
	if len(mstatus.Errors) > 0 {
		printErrors(mstatus.Errors)
	} else {
		if mstatus.Id == 0 {
			fmt.Println("\nThe server returned an error.  Are you sure that migration ID exists?")
		} else {
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
	}
}

func printImportResults(nimport newImport) {
	fmt.Println(nimport)
	if len(nimport.Errors) > 0 {
		printErrors(nimport.Errors)
	} else if nimport.Error != "" {
		fmt.Printf("\n[-] Error: %s\n", nimport.Error)
	} else {
		fmt.Printf("\n[+] Migration ID is %d\n", nimport.MigrationId)
	}
}

func printErrors(errors []apiError) {
	fmt.Println("\n[-] Errors encountered:")
	for _, err := range errors {
		fmt.Println(err)
	}
}

func printHelp() {
	fmt.Println(`
-- Outcomes Import Tool (OIT) --

The Outcomes Import Tool (OIT) can be used to easily schedule the import of outcomes
from Academic Benchmark into Canvas LMS.

At this time, only site administrators have permission to do this.  There are various
technical reasons for this.  If you have questions, or would like to have outcomes
imported into your account, please contact Instructure support or your customer
service representative.

To install:

    go get github.com/FreedomBen/outcomes-import-tool
    go install outcomes-import-tool

**This is not an officially supported tool by Instructure**

Usage is simple.  You must provide the tool with a Canvas API key, and then tell it
what to do.  The default action is to check the status of the most recent import.
OIT knows the Migration ID of the most recent import because it saves it in a json
file located at $HOME/outcomes-import-tool.json.

You must also provide it with a Canvas domain.  For a school that has
"<school-name>.instructure.com", you can simply provide the school name.  You can also
simply say "localhost" if you have a local development server running on port 3000.
The domain only needs to be passed the first time you use the tool, or when you want
to change domains.  OIT remembers the last domain automatically for you.

Once you have queried the available GUIDs, they will be stored in the aforementioned
json file.  This greatly speeds up import requests when requested by name instead of
GUID.  It also makes it possible to schedule an import by name when offline or on a
non-whitelisted IP address (such as when conducting local testing).

Example to check status:

    outcomes-import-tool --apikey="MyKey" --domain localhost

Example to check status with specified ID of 35 (which becomes the new default)

    outcomes-import-tool --apikey="MyKey" --status 35

Example to import a GUID.  This can be specified by Title from the list of available
GUIDs, or by GUID itself.  By title for Iowa standards:

    outcomes-import-tool --apikey="MyKey" --guid "Iowa"

By GUID:

    outcomes-import-tool --apikey="MyKey" --guid "A832FC24-901A-11DF-A622-0C319DFF4B22"

Example to list available GUIDs and their Titles:

    outcomes-import-tool --apikey="MyKey" --available

If you want, you can put your API key in the json file and you won't have to specify
it each time.  Be advised though, *this file is stored in plain-text in your home
directory*.  Use this for test instances of Canvas, but *it is not safe to do so with
a production system key*.`)
}
