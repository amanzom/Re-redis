package core

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/amanzom/re-redis/pkg/logger"
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
	return nil, 0, errors.New("invalid expression for resp decoding")
}

func decode(expression []byte) (interface{}, error) {
	result := make([]interface{}, 0)
	index := 0
	for index < len(string(expression)) {
		smallResult, delta, err := decodeSmall(expression[index:])
		if err != nil {
			// error could be ignored if we want to read partial cmds in case of pipelining
			logger.Error("error decoding expression, err: %v", err)
			return nil, errors.New(fmt.Sprintf("error decoding expression, err: %v", err))
		}
		result = append(result, smallResult)
		index += delta
	}
	return result, nil
}

func encodeAsBulkString(str string) []byte {
	return []byte(fmt.Sprintf("$%v\r\n%v\r\n", len(str), str))
}

// isSimple - to decide if the string value needs to be encoded as a simple string or bulk string
func encode(value interface{}, isSimple bool) []byte {
	switch v := value.(type) {
	case string:
		if isSimple {
			return []byte(fmt.Sprintf("+%v\r\n", v))
		}
		return encodeAsBulkString(v)
	case int, int8, int64, int32:
		return []byte(fmt.Sprintf(":%v\r\n", v))
	case error:
		return []byte(fmt.Sprintf("-%v\r\n", v))
	case []string: // encoded as array of bulk strings
		var b []byte
		buf := bytes.NewBuffer(b)
		for _, str := range v {
			buf.Write(encodeAsBulkString(str))
		}
		return []byte(fmt.Sprintf("*%v\r\n%v", len(v), string(buf.Bytes())))
	default:
		logger.Error("invalid value provided for resp encoding")
		return []byte(resp_nil)
	}
}
