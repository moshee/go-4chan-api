package api

import (
	"os"
	"regexp"
	"testing"
)

func maketry(t *testing.T) func(error) {
	return func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func makeassert(t *testing.T) func(bool, string) {
	return func(expr bool, desc string) {
		if !expr {
			t.Fatal("Failed:", desc)
		}
	}
}

func TestParseThread(t *testing.T) {
	try := maketry(t)
	assert := makeassert(t)

	file, err := os.Open("example.json")
	try(err)
	defer file.Close()

	thread, err := ParseThread(file, "ck")
	try(err)

	assert(thread.OP.Name == "Anonymous", "OP's name should be Anonymous")
	assert(thread.Id() == 3856791, "Thread id should be 3856791")
	assert(thread.OP.File != nil, "OP post should have a file")
	assert(len(thread.Posts) == 38, "Thread should have 38 posts")
	assert(thread.OP.ImageURL() == "http://images.4chan.org/ck/src/1346968817055.jpg", "Image URL should be 'http://images.4chan.org/ck/src/1346968817055.jpg'")
	thumbURL := thread.OP.ThumbURL()
	matched, _ := regexp.MatchString(`http://\d\.thumbs\.4chan\.org/ck/thumb/1346968817055s\.jpg`, thumbURL)
	assert(matched, "Thumb URL should match 'http://\\d.thumbs.4chan.org/ck/thumb/1346968817055s.jpg' (got '"+thumbURL+"')")
}

func TestGetIndex(t *testing.T) {
	try := maketry(t)
	assert := makeassert(t)

	threads, err := GetIndex("a", 0)
	try(err)
	assert(len(threads) > 0, "Threads should exist")
}

func TestGetThreads(t *testing.T) {
	try := maketry(t)
	n, err := GetThreads("a")
	try(err)
	for _, q := range n {
		for _, p := range q {
			if p == 0 {
				t.Fatal("There are #0 posts")
			}
		}
	}
}
