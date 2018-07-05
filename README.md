A 4chan API client for Go.

[![Build Status](https://travis-ci.org/moshee/go-4chan-api.svg?branch=master)](https://travis-ci.org/moshee/go-4chan-api) [![GoDoc](https://godoc.org/github.com/moshee/go-4chan-api/api?status.png)](https://godoc.org/github.com/moshee/go-4chan-api/api)

Supports:

- API revision [830712e on Apr 27 2018](https://github.com/4chan/4chan-API)
	* Single thread
	* Thread index
	* Board list
	* Board catalog
	* Thread list
- HTTPS
- Rate limiting
- `If-Modified-Since`
- In-place thread updating

Pull requests welcome.

#### To do

- More useful `Thread` and `*Post` methods
- Update & add more tests
- ...
