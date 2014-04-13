package api

import (
	"os"
	"testing"
)

func try(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func assert(t *testing.T, expr bool, desc string) {
	if !expr {
		t.Fatal("Failed:", desc)
	}
}

func TestParseThread(t *testing.T) {
	file, err := os.Open("example.json")
	try(t, err)
	defer file.Close()

	thread, err := ParseThread(file, "ck")
	try(t, err)

	assert(t, thread.OP.Name == "Anonymous", "OP's name should be Anonymous")
	assert(t, thread.Id() == 3856791, "Thread id should be 3856791")
	assert(t, thread.OP.File != nil, "OP post should have a file")
	assert(t, len(thread.Posts) == 38, "Thread should have 38 posts")
	imageURL := thread.OP.ImageURL()
	assert(t, imageURL == "http://i.4cdn.org/ck/src/1346968817055.jpg", "Image URL should be 'http://i.4cdn.org/ck/src/1346968817055.jpg' (got '"+imageURL+"')")
	thumbURL := thread.OP.ThumbURL()
	assert(t, thumbURL == "http://t.4cdn.org/ck/thumb/1346968817055s.jpg", "Thumb URL should be 'http://t.4cdn.org/ck/thumb/1346968817055s.jpg' (got '"+thumbURL+"')")
}

func TestGetIndex(t *testing.T) {
	threads, err := GetIndex("a", 0)
	try(t, err)
	assert(t, len(threads) > 0, "Threads should exist")
}

func TestGetThreads(t *testing.T) {
	n, err := GetThreads("a")
	try(t, err)
	for _, q := range n {
		for _, p := range q {
			if p == 0 {
				t.Fatal("There are #0 posts")
			}
		}
	}
}
