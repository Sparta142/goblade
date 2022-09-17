//go:build !(windows && amd64)

package oodle

import "errors"

var ErrPlatformNotSupported = errors.New("oodle: platform not supported")

func Decode(payload []byte, rawLen uint32) ([]byte, error) {
	return nil, ErrPlatformNotSupported
}
