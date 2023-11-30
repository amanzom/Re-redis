package resp_test

import (
	"fmt"
	"testing"

	"github.com/amanzom/re-redis/core/resp"
)

func TestSimpleStringDecode(t *testing.T) {
	cases := map[string]string{
		"+OK\r\n": "OK",
	}
	for c, v := range cases {
		value, _ := resp.Decode([]byte(c))
		if value != v {
			t.Fail()
		}
	}
}

func TestSimpleErrorDecode(t *testing.T) {
	cases := map[string]string{
		"-Error Message\r\n": "Error Message",
	}
	for c, v := range cases {
		value, _ := resp.Decode([]byte(c))
		if value != v {
			t.Fail()
		}
	}
}

func TestInt64Decode(t *testing.T) {
	cases := map[string]int64{
		":145\r\n": 145,
		":0\r\n":   0,
	}
	for c, v := range cases {
		value, _ := resp.Decode([]byte(c))
		if value != v {
			t.Fail()
		}
	}
}

func TestBulkStringDecode(t *testing.T) {
	cases := map[string]string{
		"$5\r\nhello\r\n": "hello",
		"$0\r\n\r\n":      "",
	}

	for c, v := range cases {
		value, _ := resp.Decode([]byte(c))
		if value != v {
			t.Fail()
		}
	}
}

func TestArrayDecode(t *testing.T) {
	cases := map[string][]interface{}{
		"*0\r\n":                                                   {},
		"*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n":                     {"hello", "world"},
		"*3\r\n:1\r\n:2\r\n:3\r\n":                                 {int64(1), int64(2), int64(3)},
		"*5\r\n:1\r\n:2\r\n:3\r\n:4\r\n$5\r\nhello\r\n":            {int64(1), int64(2), int64(3), int64(4), "hello"},
		"*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n": {[]int64{int64(1), int64(2), int64(3)}, []interface{}{"Hello", "World"}},
	}

	for c, expectedArr := range cases {
		value, _ := resp.Decode([]byte(c))
		resultArr, ok := value.([]interface{})
		if !ok {
			t.Fail()
		}
		if len(resultArr) != len(expectedArr) {
			t.Fail()
		}

		for i := range resultArr {
			if fmt.Sprintf("%v", resultArr[i]) != fmt.Sprintf("%v", expectedArr[i]) {
				t.Fail()
			}
		}
	}
}
