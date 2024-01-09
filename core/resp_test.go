package core

import (
	"fmt"
	"testing"
)

func TestSimpleStringDecode(t *testing.T) {
	cases := map[string][]string{
		"+OK\r\n": {"OK"},
	}
	for c, v := range cases {
		valuesInterface, _ := Decode([]byte(c))
		values := valuesInterface.([]interface{})
		for i := 0; i < len(values); i++ {
			if values[i].(string) != v[i] {
				t.Fail()
			}
		}
	}
}

func TestSimpleErrorDecode(t *testing.T) {
	cases := map[string][]string{
		"-Error Message\r\n": {"Error Message"},
	}
	for c, v := range cases {
		valuesInterface, _ := Decode([]byte(c))
		values := valuesInterface.([]interface{})
		for i := 0; i < len(values); i++ {
			if values[i].(string) != v[i] {
				t.Fail()
			}
		}
	}
}

func TestInt64Decode(t *testing.T) {
	cases := map[string][]int64{
		":145\r\n": {145},
		":0\r\n":   {0},
	}
	for c, v := range cases {
		valuesInterface, _ := Decode([]byte(c))
		values := valuesInterface.([]interface{})
		for i := 0; i < len(values); i++ {
			if values[i].(int64) != v[i] {
				t.Fail()
			}
		}
	}
}

func TestBulkStringDecode(t *testing.T) {
	cases := map[string][]string{
		"$5\r\nhello\r\n$5\r\nhello\r\n": {"hello", "hello"},
		"$5\r\nhello\r\n":                {"hello"},
		"$0\r\n\r\n":                     {""},
	}

	for c, v := range cases {
		valuesInterface, _ := Decode([]byte(c))
		values := valuesInterface.([]interface{})
		for i := 0; i < len(values); i++ {
			if values[i].(string) != v[i] {
				t.Fail()
			}
		}
	}
}

// for testing pipelining: printf "*1\r\n\$4\r\nPING\r\n*3\r\n\$3\r\nSET\r\n\$1\r\nk\r\n\$1\r\nv\r\n*2\r\n\$3\r\nGET\r\n\$1\r\nk\r\n" | nc localhost 7369
func TestArrayDecode(t *testing.T) {
	cases := map[string][][]interface{}{
		"*0\r\n":                               {{}},
		"*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n": {{"hello", "world"}},
		"*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n": {{"hello", "world"}, {"hello", "world"}},
		"*3\r\n:1\r\n:2\r\n:3\r\n":                                                                    {{int64(1), int64(2), int64(3)}},
		"*5\r\n:1\r\n:2\r\n:3\r\n:4\r\n$5\r\nhello\r\n":                                               {{int64(1), int64(2), int64(3), int64(4), "hello"}},
		"*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n":                                    {{[]int64{int64(1), int64(2), int64(3)}, []interface{}{"Hello", "World"}}},
		"*2\r\n$7\r\nCOMMAND\r\n$4\r\nDOCS\r\n":                                                       {{"COMMAND", "DOCS"}},
		"*1\r\n$4\r\nPING\r\n*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n*2\r\n$3\r\nGET\r\n$1\r\nk\r\n": {{"PING"}, {"SET", "k", "v"}, {"GET", "k"}},
	}

	for c, expectedArr := range cases {
		value, _ := Decode([]byte(c))
		resultArrInterface, ok := value.([]interface{})
		if !ok {
			t.Fail()
		}

		if len(resultArrInterface) != len(expectedArr) {
			t.Fail()
		}

		for ind, resultInterface := range resultArrInterface {
			r := resultInterface.([]interface{})
			if len(r) != len(expectedArr[ind]) {
				t.Fail()
			}

			for i := range r {
				if fmt.Sprintf("%v", r[i]) != fmt.Sprintf("%v", expectedArr[ind][i]) {
					t.Fail()
				}
			}
		}
	}
}
