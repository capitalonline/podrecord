package utils

import (
	"fmt"
	"testing"
)

func TestRound(t *testing.T) {
	fmt.Println(fmt.Sprintf("%.2f", Round(10.2438, 2)))
	fmt.Println(Round(10.2438, 2))
}
