package oodle

/*
#cgo windows LDFLAGS: -ldbghelp -lpsapi
#cgo windows CFLAGS: --std=c17 -Wall -Wextra -Wno-unused-variable

#include <minwindef.h>
#include <stdbool.h>
#include <stdint.h>

DWORD setup(const LPCSTR lpLibFileName);
void shutdown();
bool decode(const void *comp, const int64_t compLen, void* raw, const int64_t rawLen);
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

var (
	ErrDecompressionFailed = errors.New("oodle: decompression failed in native code")
	ErrSetupFailed         = errors.New("oodle: setup failed in native code")
)

func Decode(comp, raw []byte) error {
	if C.decode(unsafe.Pointer(&comp[0]), C.longlong(len(comp)), unsafe.Pointer(&raw[0]), C.longlong(len(raw))) {
		return nil
	}

	return ErrDecompressionFailed
}

func Setup() error {
	// Get the location of the game executable
	exe, err := findGameExe()
	if err != nil {
		return fmt.Errorf("oodle setup: %w", err)
	}

	log.Debugf("Loaded game executable at: %s", exe)

	// Pass the exe path to native code for setup
	cstr := C.CString(exe)
	defer C.free(unsafe.Pointer(cstr))

	if status := C.setup(cstr); status != 0 {
		return fmt.Errorf("%w (status %d)", ErrSetupFailed, status)
	}

	return nil
}

func Shutdown() {
	C.shutdown()
}
