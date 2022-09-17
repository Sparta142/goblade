package oodle

/*
#cgo windows LDFLAGS: -ldbghelp

#include <minwindef.h>
#include <stdbool.h>
#include <stdlib.h>

DWORD init(const char *lpLibFileName);
void deinit();
void *decode(void *comp, __int64 compLen, __int64 rawLen);
*/
import "C"

import (
	"errors"
	"os"
	"sync"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

var (
	ErrExeNotFound         = errors.New("oodle: game executable not found")
	ErrDecompressionFailed = errors.New("oodle: decompression failed in native code")
)

// The game executable filename (and process name).
const exeName = "ffxiv_dx11.exe"

// Candidate paths for the game executable.
var exePaths = [...]string{
	"${ProgramFiles(x86)}\\SquareEnix\\FINAL FANTASY XIV - A Realm Reborn\\game\\" + exeName,     // Square Enix
	"${ProgramFiles(x86)}\\Steam\\steamapps\\common\\FINAL FANTASY XIV Online\\game\\" + exeName, // Steam
}

var oodleLock sync.Mutex

func Decode(payload []byte, rawLen uint32) ([]byte, error) {
	oodleLock.Lock()
	defer oodleLock.Unlock()

	// Marshal the compressed payload into C memory
	comp := C.CBytes(payload)
	defer C.free(comp)

	// Decompress it
	raw := C.decode(comp, C.longlong(len(payload)), C.longlong(rawLen))
	if raw == unsafe.Pointer(nil) {
		return nil, ErrDecompressionFailed
	}
	defer C.free(raw)

	// Marshal the decompressed data back into Go memory and return it
	return C.GoBytes(raw, C.int(rawLen)), nil
}

func findGameExe() (string, error) {
	for _, path := range exePaths {
		path = os.ExpandEnv(path)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// TODO: Scan the process list for the game process
	// If we find it, then save its filename somewhere
	// so we can use it as a hint in the future
	return "", ErrExeNotFound
}

func init() {
	oodleLock.Lock()
	defer oodleLock.Unlock()

	// Get the location of the game executable
	exe, err := findGameExe()
	if err != nil {
		panic(err) // TODO
	}

	log.Debugf("Loaded game executable at: %s", exe)

	sss := C.CString(exe)
	defer C.free(unsafe.Pointer(sss))

	if status := C.init(sss); status != 0 {
		panic("failed to init from game executable") // TODO
	}
}
