package api

import (
	"os"
	"testing"
	"fmt"
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

	fmt.Println(thread.Posts[0])
	assert(thread.OP.Name == "Anonymous", "OP's name should be Anonymous")
	assert(thread.Id() == 3856791, "Thread id should be 3856791")
	assert(thread.OP.File != nil, "OP post should have a file")
	assert(len(thread.Posts) == 38, "Thread should have 38 posts")
}

func TestGetIndex(t *testing.T) {
	try := maketry(t)
	assert := makeassert(t)

	threads, err := GetIndex("a", 0)
	try(err)
	fmt.Println(threads[0].OP)
	assert(len(threads) > 0, "Threads should exist")
}
