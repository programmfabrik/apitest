package util

/*
aliases for possible Json Types; consider embedding the types like

type genericJsonImpl = interface{}
type GenericJson struct {
	genericJsonImpl
}

so that we can have GenericJson as a receiver for methods, alas .. unmarshalling is problematic
*/

type GenericJson = interface{}
type JsonObject = map[string]GenericJson
type JsonArray = []GenericJson
type JsonString = string
type JsonNumber = float64
type JsonBool = bool
