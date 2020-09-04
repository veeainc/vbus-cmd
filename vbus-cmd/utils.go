package main

import vBus "bitbucket.org/vbus/vbus.go"


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
