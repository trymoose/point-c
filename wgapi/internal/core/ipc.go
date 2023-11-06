package core

import "fmt"

type KeyValue interface {
	Key() string
	fmt.Stringer
}
