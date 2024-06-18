package main

import (
	"fmt"
	"testing"
)

func TestGenerateData(test *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Println(generateData())
	}
}
