import requests, sys, getopt, json

def printPrettyMigrationStatus(data):
    if data['id'] == None or data['id'] == '' or data['id'] == 0:
        print '[-] The server returned an error.  Are you sure that migration ID exists?'
    else:
        print '[+] Migration status for Migration: [ ' + data['id'] + ' ]:'
        print '  - Workflow State: ' + data['workflow_state']
        print '  - Migration Issues Count: ' + data['migration_issues_count']
        print '  - Migration Issues: '
        for issue in data['migration_issues']:
            print '    - ID: ' + issue['id']
            print '    - Link: ' + issue['error_report_html_url']
            print '    - Issue Type: ' + issue['issue_type']
            print '    - Error Message: ' + issue['error_message']
            print '    - Description: ' + issue['description']

def printPrettyImportResults(data):
    """
    if len(nimport.Errors) > 0 {
		printErrors(nimport.Errors)
	} else if nimport.Error != "" {
		fmt.Printf("\nError: %s\n", nimport.Error)
	} else {
		fmt.Printf("\nMigration ID is %d\n", nimport.MigrationId)
	}
    """
    if len(data['errors']) > 0:
        print '[-] Errors were run into. Check Response File.'
    else:
        print '[+] Migration ID is: [ ' + data['migration_id'] + ' ]'

def normalizeAndValidateDomain(domain):
    if domain == '':
        print '[-] No Domain Provided.... Failing...'
        sys.exit(1)
    elif domain == 'localhost':
        domain = 'http://localhost:3000'
    else:
        if domain.endswith('/'):
            domain = domain[:domain.rindex('/')]
        if not domain.startsWith('https://'):
            print '[-] Isn\'t a valid Canvas Domain as must be run through https.'
            sys.exit()
        else:
            print '[+] Verifying SSL Cert of Domain.....'
            try:
                requests.get(domain, verify = True)
            except requests.exceptions.SSLError:
                print '[-] SSL Cert isn\'t valid.'
                sys.exit(1)
            print '[+] Verified.'
    return domain

def performAvailable(domain, token):
    print '[+] Requesting All Available Outcomes...'
    r = requests.get(domain + "/api/v1/global/outcomes_import/available?access_token=" + token)
    f = file.open('available-response.json')
    f.write(r.json())
    f.close()
    print r.json()

def performStatus(domain, token, value):
    print '[+] Requesting Migration Status for: [ ' + value + ' ]'
    r = requests.get(domain + '/api/v1/global/outcomes_import/migration_status/' + value + '?access_token=' + token)
    f = file.open('migration-status-' + value + '-response.json')
    f.write(r.json())
    f.close()
    printPrettyMigrationStatus(r.json())
    print '[+] Done.'

def performGUID(domain, token, value):
    headers = {'Authorization':('Bearer ' + token)}
    payload = {'guid' : value}
    print '[+] Requesting import of GUID: ' + value
    r = requests.post((domain + ('/api/v1/global/outcomes_import')), headers=headers, data=payload)
    f = file.open('import-of-' + value + '-response.json')
    f.write(r.json())
    f.close()
    printPrettyImportResults(r.json())
    print '[+] Done.'

def main(argv):
    configFile = ''
    domain = ''
    token = ''
    mode = ''
    modeValue = None
    try:
        opts, args = getopt.getopt(argv,"hc:d:t:s:ag:")
    except getopt.GetoptError:
        print 'outcomes_import_tool.py (-c configFile | (-d domain & -t token)) (-s | -a | -g )'
        sys.exit(2)
    for opt, arg in opts:
       if opt == '-h':
          print 'outcomes_import_tool.py (-c configFile | (-d domain & -t token)) (-s | -a | -g )'
          sys.exit()
       elif opt in ('-c', '-config'):
          configFile = arg
      elif opt in ('-d', '-domain'):
          domain = arg
      elif opt in ('-t', '-token'):
          token = arg
      elif opt in ('-s', '-status'):
          mode = 'status'
          modeValue = arg
      elif opt in ('-a', '-available'):
          mode = 'available'
      elif opt in ('-g', '-guid'):
          mode = 'guid'
          modeValue = arg
  if configfile != '':
      with open(configfile) as data_file:
          data = json.load(data_file)
          if data['token'] != None or data['token'] != '':
              token = data['token']
          if data['domain'] != None or data['domain'] != '':
              domain = data['domain']
  domain = normalizeAndValidateDomain(domain)
  if token == '':
      print '[-] No Token Provided.... Failing...'
      sys.exit(1)
  if mode == 'guid':
      print '[+] Performing GUID....'
      performGUID(domain, token, modeValue)
  elif mode == 'status':
      print '[+] Performing Status....'
      performStatus(domain, token, modeValue)
  else:
      print '[+] Performing Available...'
      performAvailable(domain, token)

if __name__ == '__main__':
    main(sys.argv[1:])
