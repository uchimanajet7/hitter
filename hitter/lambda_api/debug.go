package main

import (
	"log"
)

// refer to:
// https://golang.org/src/net/mail/message.go?h=debugT#L36
var debug = debugT(false)

type debugT bool

func (d debugT) Printf(format string, args ...interface{}) {
	if d {
		log.Printf("[DEBUG] "+format, args...)
	}
}
