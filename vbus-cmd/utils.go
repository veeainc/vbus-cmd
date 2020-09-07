package main

import (
	vBus "bitbucket.org/vbus/vbus.go"
	"strings"
)


type JsonArray = []interface{}

// Check if an interface{} is a map and contains the provided key.
func hasKey(obj interface{}, key string) bool {
	if m, ok := obj.(vBus.JsonObj); ok {
		if _, found := m[key]; found {
			return true
		}
	}
	return false
}

func getKey(obj interface{}, key string) vBus.JsonAny {
	if m, ok := obj.(vBus.JsonObj); ok {
		if k, found := m[key]; found {
			return k
		}
	}
	return nil
}

// badSubject will do quick test on whether a subject is acceptable.
// Spaces are not allowed and all tokens should be > 0 in len.
func badSubject(subj string) bool {
	if strings.ContainsAny(subj, " \t\r\n") {
		return true
	}
	tokens := strings.Split(subj, ".")
	for _, t := range tokens {
		if len(t) == 0 {
			return true
		}
	}
	return false
}