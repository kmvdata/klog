package klog

import (
	"testing"
	"time"
)

func TestHelloWorld(t *testing.T) {
	// Init
	InitKLog("./log/my-log", 0, true)

	// Set Max Size
	// Or klog.SetMaxFileSizeMB(200) ==> Set to 200MB
	SetMaxFileSizeKB(500)

	// Test Code
	for i := 0; i < 10000; i++ {
		Info.Printf("Hello %s", "World!")
		Error.Printf("Hello %s", "Error!")
	}

	// Sleep make sure that uncompress *.log can be deleted.
	time.Sleep(time.Duration(2) * time.Second)

}