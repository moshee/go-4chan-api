A 4chan API client for Go. Supports:
- API revision 25 (22 Mar 2013)
	* Single thread
	* Thread index
	* Board list
	* Board catalog
	* Thread list
- HTTPS
- Rate limiting
- `If-Modified-Since`

[Examples and docs on GoDoc.](http://godoc.org/github.com/moshee/go-4chan-api)

Pull requests welcome.

#### To do

- Test the 1 second request cooldown
- More useful `Thread` and `*Post` methods
- Update & add more tests
- ...
