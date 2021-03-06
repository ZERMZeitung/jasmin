package main

import (
	"fmt"
	"log"
)

func Fatal(v ...interface{}) {
	msg := "[Fatal]"
	for _, s := range v {
		msg += " " + fmt.Sprint(s)
	}
	log.Fatalln(msg)
}

func Err(v ...interface{}) {
	msg := "[Err]"
	for _, s := range v {
		msg += " " + fmt.Sprint(s)
	}
	log.Println(msg)
}

func Warn(v ...interface{}) {
	msg := "[Warn]"
	for _, s := range v {
		msg += " " + fmt.Sprint(s)
	}
	log.Println(msg)
}

func Info(v ...interface{}) {
	msg := "[Info]"
	for _, s := range v {
		msg += " " + fmt.Sprint(s)
	}
	log.Println(msg)
}
