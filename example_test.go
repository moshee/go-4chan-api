package api

import (
	"fmt"
	"time"
)

func ExampleVariables() {
	// All requests will be made with HTTPS
	SSL = true

	// will be pulled up to 10 seconds when first used
	UpdateCooldown = 5 * time.Second

	// get index, threads, etc
}

func ExampleGetIndex() {
	threads, err := GetIndex("a", 0)
	if err != nil {
		panic(err)
	}
	for _, thread := range threads {
		fmt.Println(thread)
	}
}

func ExampleThread() {
	thread, err := GetThread("a", 77777777)
	if err != nil {
		panic(err)
	}
	// will block until the cooldown is reached
	thread.Update()
	fmt.Println(thread)
}
