package oodle

/*
#cgo windows LDFLAGS: -ldbghelp -lpsapi
#cgo windows CFLAGS: -Wall -Wextra

#include <minwindef.h>
#include <stdbool.h>
#include <stdint.h>

DWORD init(const LPCSTR lpLibFileName);
void deinit();
bool decode(const void *comp, const int64_t compLen, void* raw, const int64_t rawLen);
*/
import "C"

import (
	"errors"
	"os"
	"reflect"
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

func Decode(comp, raw []byte) error {
	// Get a pointer to the beginning of the slice data.
	// This is cursed, but saves us a malloc by not marshaling data into C memory.
	compHdr := (*reflect.SliceHeader)(unsafe.Pointer(&comp))
	rawHdr := (*reflect.SliceHeader)(unsafe.Pointer(&raw))

	// Decompress the slice
	success := C.decode(
		unsafe.Pointer(compHdr.Data),
		C.longlong(len(comp)),
		unsafe.Pointer(rawHdr.Data),
		C.longlong(len(raw)),
	)
	if !success {
		return ErrDecompressionFailed
	}

	return nil
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
