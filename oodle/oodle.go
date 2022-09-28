//go:build !(cgo && windows && amd64)

package oodle

import (
	"C" // To prevent "C source files not allowed when not using cgo or SWIG"
	"errors"
)

var ErrPlatformNotSupported = errors.New("oodle: platform not supported")

func Decode(payload []byte, rawLen uint32) ([]byte, error) {
	return nil, ErrPlatformNotSupported
}
