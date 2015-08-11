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
	title string `json:"title"`
	guid  string `json:"guid"`
}

type migration_status struct {
}

type new_import struct {
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
	// the old way
	// body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// log.Fatalln(err)
	// }
	// log.Println(string(body))
}

func get_status(req request, migration_id int) {
	log.Println("Retrieving status for migration")
}

func import_guid(req request, guid string) {
	log.Println("Scheduling import of guid")
}

func print_importable_guids(guids []importable_guid) {
  log.Println(guids)
  fmt.Printf("GUIDs available to import:\n\n")
  for _, guid := range guids {
    fmt.Printf("%s: %s", guid.title, guid.guid)
  }
}
