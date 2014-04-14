A 4chan API client for Go.

[![Build Status](https://travis-ci.org/moshee/go-4chan-api.svg?branch=master)](https://travis-ci.org/moshee/go-4chan-api)

Supports:

- API revision 34 (2014-04-12)
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
