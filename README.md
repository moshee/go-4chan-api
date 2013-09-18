A 4chan API client for Go. Supports:
- API revision 28 (16 September 2013)
	* Single thread
	* Thread index
	* Board list
	* Board catalog
	* Thread list
- HTTPS
- Rate limiting
- `If-Modified-Since`
- In-place thread updating

[Examples and docs on GoDoc.](http://godoc.org/github.com/moshee/go-4chan-api)

Pull requests welcome.

#### To do

- Test the 1 second request cooldown
- More useful `Thread` and `*Post` methods
- Update & add more tests
- ...
