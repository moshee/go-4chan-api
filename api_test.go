package api

import (
	"os"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}

	assert := func(expr bool, desc string) {
		if !expr {
			t.Fatal("Failed:", desc)
		}
	}

	file, err := os.Open("example.json")
	try(err)
	defer file.Close()

	thread, err := ParseReader(file)
	try(err)

	assert(thread.OP().Name == "Anonymous", "OP's name is Anonymous")
	assert(thread.Id() == 3856791, "Thread id is 3856791")
	assert(thread.OP().File != nil, "OP post has a file")
	assert(len(thread) == 38, "Thread has 38 posts")
}
