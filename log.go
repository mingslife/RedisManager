package main

import (
	"fmt"
)

const (
	DevelopmentMode = iota
	ProductionMode
)

var mode = ProductionMode

type Log struct{}

func (log *Log) Error(v interface{}) {
	if mode == DevelopmentMode {
		fmt.Println(v)
	}
}
func (log *Log) Debug(v interface{}) {
	if mode == DevelopmentMode {
		fmt.Println(v)
	}
}
