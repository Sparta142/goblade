//go:build !cgo || !(windows && amd64)

package oodle

import "errors"

var ErrPlatformNotSupported = errors.New("oodle: platform not supported")

func Decode(_, _ []byte) error {
	return ErrPlatformNotSupported
}
