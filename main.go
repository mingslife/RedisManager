package main

import (
	"fmt"
)

func main() {
	fmt.Println("Test RedisManager")
	NewRedisManager("127.0.0.1", 6379, "", 0)
}
