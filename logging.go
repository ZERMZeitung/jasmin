package main

import "log"

func Err(v ...interface{}) {
	log.Println(append([]interface{}{"[Err]"}, v...)...)
}

func Warn(v ...interface{}) {
	log.Println(append([]interface{}{"[Warn]"}, v...)...)
}

func Info(v ...interface{}) {
	log.Println(append([]interface{}{"[Info]"}, v...)...)
}
