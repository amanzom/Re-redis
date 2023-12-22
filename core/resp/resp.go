package resp

import (
	"errors"
	"fmt"
)

func readSimpleString(expression []byte) (string, int, error) {
	for pos := 1; pos < len(expression); pos++ {
		if expression[pos] == '\r' {
			return string(expression[1:pos]), pos + 2, nil
		}
	}
	return "", 0, errors.New("invalid resp encoded string")
}

func readSimpleError(expression []byte) (string, int, error) {
	return readSimpleString(expression)
}

// add negative integers support if needed
func readInt64(expression []byte) (int64, int, error) {
	var val int64 = 0
	for pos := 1; pos < len(expression); pos++ {
		if expression[pos] == '\r' {
			return val, pos + 2, nil
		}
		if expression[pos] < '0' || expression[pos] > '9' {
			return 0, 0, errors.New("invalid resp encoded string")
		}
		val = val*10 + int64(expression[pos]-'0')
	}
	return 0, 0, errors.New("invalid resp encoded string")
}

func readLength(expression []byte) (int64, int, error) {
	var length int64 = 0
	for pos := 1; pos < len(expression); pos++ {
		if expression[pos] == '\r' {
			return length, pos + 2, nil
		}
		if expression[pos] < '0' || expression[pos] > '9' {
			return 0, 0, errors.New("invalid resp encoded string")
		}
		length = length*10 + int64(expression[pos]-'0')
	}
	return 0, 0, errors.New("invalid resp encoded string")
}

func readBulkString(expression []byte) (string, int, error) {
	length, delta, err := readLength(expression)
	if err != nil {
		return "", 0, err
	}
	return string(expression[delta : delta+int(length)]), delta + int(length) + 2, nil
}

func readArray(expression []byte) ([]interface{}, int, error) {
	length, delta, err := readLength(expression)
	if err != nil {
		return nil, 0, err
	}

	pos := delta
	var elems []interface{}
	for i := 0; i < int(length); i++ {
		elem, deltaSmall, err := decodeSmall(expression[pos:])
		if err != nil {
			return nil, 0, err
		}
		elems = append(elems, elem)
		pos += deltaSmall
	}
	return elems, pos, nil
}

// returns the req resp decoded object, delta(pos(not index) upto which elements have been read) and error
func decodeSmall(expression []byte) (interface{}, int, error) {
	if len(expression) == 0 {
		return nil, 0, errors.New("empty expression for resp decoding")
	}

	switch expression[0] {
	case '+':
		return readSimpleString(expression)
	case '-':
		return readSimpleError(expression)
	case ':':
		return readInt64(expression)
	case '$':
		return readBulkString(expression)
	case '*':
		return readArray(expression)
	}
	return nil, 0, nil
}

func Decode(expression []byte) (interface{}, error) {
	result, _, err := decodeSmall(expression)
	return result, err
}

// isSimple - to decide if the string value needs to be encoded as a simple string or bulk string
func Encode(value interface{}, isSimple bool) ([]byte, error) {
	switch value.(type) {
	case string:
		v := value.(string)
		if isSimple {
			return []byte(fmt.Sprintf("+%v\r\n", v)), nil
		}
		return []byte(fmt.Sprintf("$%v\r\n%v\r\n", len(v), v)), nil
	case int64:
		v := value.(int64)
		return []byte(fmt.Sprintf(":%v\r\n", v)), nil
	}
	return nil, errors.New("invalid type provided for resp encoding")
}
