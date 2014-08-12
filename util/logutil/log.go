package logutil

import "log"

func ErrorFatal(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func ErrorNormal(msg string, err error) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}
