package logger

import "fmt"

func Info(s string, v ...any) {
	fmt.Println(fmt.Sprintf(s, v...))
}

func Error(s string, v ...any) {
	fmt.Println(fmt.Errorf(s, v...))
}
