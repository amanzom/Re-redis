package core

import (
	"strconv"
)

// Note : Type represents datatype in which the object is actually stored, encoding represents
// datatype in which the object needs to be interpreted

// typeEncoding - 1st 4 bits represents the type, last 4 bits represents the encoding
// for example for storing an string object type encoding as int  - type: ObjectTypeString(0 - 0000), encoding: ObjectEncodingInt(1 - 0001),
// TypeEncoding in this case - 00000001
type typeEncoding uint8

const (
	// types
	ObjectTypeString uint8 = 0

	// encodings
	ObjectEncodingRaw            uint8 = 0 // when object value bytes size > 44 - then its encoded as Raw string otherwise as embedded string
	ObjectEncodingInt            uint8 = 1
	ObjectEncodingEmbeddedString uint8 = 8
)

// getting first 4 bits and setting last 4 bits as 0000
func getType(typeEncoding uint8) uint8 {
	return (typeEncoding >> 4) << 4
}

// getting last 4 bits, and setting 1st 4 bits as 0000
func getEncoding(typeEncoding uint8) uint8 {
	return typeEncoding & 0b00001111
}

// checks if type t matches te
func assertType(t uint8, te uint8) bool {
	return getType(t) == te
}

// checks if encoding e matches ee
func assertEncoding(e uint8, ee uint8) bool {
	return getEncoding(e) == ee
}

// currently we support only type string and its corresponding encodings - int, raw, embedded
// TODO: add support for deducing other types and encoding when introduced
func deduceTypeEncoding(v string) (uint8, uint8) {
	oType := ObjectTypeString
	if _, err := strconv.ParseInt(v, 10, 64); err == nil {
		return oType, ObjectEncodingInt
	}
	if len(v) <= 44 {
		return oType, ObjectEncodingEmbeddedString
	}
	return oType, ObjectEncodingRaw
}
