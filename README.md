# Outcomes Import Tool (OIT)

The Outcomes Import Tool (OIT) can be used to easily schedule the import of outcomes from Academic Benchmark into [Canvas LMS](https://github.com/instructure/canvas-lms).  At this time, only site administrators have permission to do this.  There are various technical reasons for this.  If you have questions, or would like to have outcomes imported into your account, please contact Instructure support or your customer service representative.

To Download:

Choose the appropriate pre-built binary:

- [Linux](https://github.com/FreedomBen/outcomes-import-tool-bins/blob/master/linux/0.0.1/outcomes-import-tool?raw=true)
- [OS X](https://github.com/FreedomBen/outcomes-import-tool-bins/blob/master/osx/0.0.1/outcomes-import-tool?raw=true)
- [Windows](https://github.com/FreedomBen/outcomes-import-tool-bins/blob/master/windows/0.0.1/outcomes-import-tool.exe?raw=true)

*Older versions are available at https://github.com/FreedomBen/outcomes-import-tool-bins.  These are all 64-bit binaries.*

To install from source (assuming your Go environment is already set up):

    go get github.com/FreedomBen/outcomes-import-tool
    go install outcomes-import-tool

**This is not an officially supported tool by Instructure**

Usage is simple.  You must provide the tool with a [Canvas API key](https://canvas.instructure.com/doc/api/file.oauth.html), and then tell it what to do.  The default action is to check the status of the most recent import.  OIT knows the Migration ID of the most recent import because it saves it in a json file located at `$HOME/outcomes-import-tool.json`.

You must also provide it with a Canvas domain.  For a school that has "<school-name>.instructure.com", you can simply provide the school name.  You can also simply say "localhost" if you have a local development server running on port 3000.  The domain only needs to be passed the first time you use the tool, or when you want to change domains.  OIT remembers the last domain automatically for you.

Once you have queried the available GUIDs, they will be stored in the aforementioned json file.  This greatly speeds up import requests when requested by name instead of GUID.  It also makes it possible to schedule an import by name when offline or on a non-whitelisted IP address (such as when conducting local testing).

Example to check status:

    outcomes-import-tool --apikey="MyKey" --domain localhost

Example to check status with specified ID of 35 (which becomes the new default)

    outcomes-import-tool --apikey="MyKey" --status 35

Example to import a GUID.  This can be specified by Title from the list of available GUIDs, or by GUID itself.  By title for Iowa standards:

    outcomes-import-tool --apikey="MyKey" --guid "Iowa"

By GUID:

    outcomes-import-tool --apikey="MyKey" --guid "A832FC24-901A-11DF-A622-0C319DFF4B22"

Example to list available GUIDs and their Titles:

    outcomes-import-tool --apikey="MyKey" --available

If you want, you can put your API key in the json file and you won't have to specify it each time.  Be advised though, *this file is stored in plain-text in your home directory*.  Use this for test instances of Canvas, but *it is not safe to do so with a production system key*.
