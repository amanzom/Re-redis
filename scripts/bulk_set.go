package main

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"

	"github.com/amanzom/re-redis/core"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:7369")
	if err != nil {
		panic(err)
	}

	for {
		k, v := getRandomKeyValue()
		cmd := fmt.Sprintf("SET %s %d", k, v)
		fmt.Println(cmd)
		_, err = conn.Write(core.Encode(strings.Split(cmd, " "), false))
		if err != nil {
			panic(err)
		}

		var buf [512]byte
		_, err = conn.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				return
			}
			panic(err)
		}
	}
	conn.Close()
}

func getRandomKeyValue() (string, int64) {
	value := int64(rand.Uint64() % 5000000)
	return "k" + strconv.FormatInt(value, 10), value
}
