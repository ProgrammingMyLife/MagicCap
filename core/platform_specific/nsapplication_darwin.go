// +build darwin
// This code is a part of MagicCap which is a MPL-2.0 licensed project.
// Copyright (C) Jake Gealer <jake@gealer.email> 2019.

package platformspecific

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "nsapplication_darwin.h"
*/
import "C"

// OnReady is a function that will be called by the C ready function.
var OnReady func()

// CReadyCallback is the callback which will be called by C.
//export CReadyCallback
func CReadyCallback() {
	OnReady()
}

// NSApplicationStart exports a function which is used to start the NSApplication instance.
func NSApplicationStart(ReadyCallback func()) func() {
	OnReady = ReadyCallback
	return func() {
		C.DelegateInit()
	}
}
