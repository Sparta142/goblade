package oodle

import (
	"errors"
	"fmt"
	"os"
	"path"
	"unsafe"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

var (
	ErrExeNotFound     = errors.New("oodle: game executable not found")
	ErrProcessNotFound = errors.New("oodle: game process not found")
)

// The game executable filename (and process name).
const exeName = "ffxiv_dx11.exe"

// Templated candidate paths for the game executable.
var exePaths = [...]string{
	"${ProgramFiles(x86)}\\SquareEnix\\FINAL FANTASY XIV - A Realm Reborn\\game\\" + exeName,     // Square Enix
	"${ProgramFiles(x86)}\\Steam\\steamapps\\common\\FINAL FANTASY XIV Online\\game\\" + exeName, // Steam
}

// Gets the absolute path of the game's executable.
func findGameExe() (string, error) {
	if filename, err := getProcess(exeName); err == nil {
		log.Infof("Game is currently running from %s", filename)
		return filename, nil
	}

	log.Infof("No process named %s, trying hard-coded paths", exeName)

	for _, path := range exePaths {
		path = os.ExpandEnv(path)

		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			log.Infof("Found matching candidate: %s", path)
			return path, nil
		}

		log.Infof("Candidate not found: %s", path)
	}

	log.Warn("Game process not running and no candidates match")

	return "", ErrExeNotFound
}

func getProcess(name string) (string, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", fmt.Errorf("create snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot) //nolint:errcheck

	entry := &windows.ProcessEntry32{
		Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{})),
	}
	if err = windows.Process32First(snapshot, entry); err != nil {
		return "", fmt.Errorf("get process entry: %w", err)
	}

	// Loop through all process entries until we find
	// one with an image name matching `name`.
	for {
		if path.Base(windows.UTF16ToString(entry.ExeFile[:])) == name {
			return getProcessFilename(entry)
		} else if windows.Process32Next(snapshot, entry) != nil {
			return "", ErrProcessNotFound
		}
	}
}

func getProcessFilename(entry *windows.ProcessEntry32) (string, error) {
	// Open a limited-use handle to the process
	proc, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
	if err != nil {
		return "", fmt.Errorf("open process: %w", err)
	}
	defer windows.CloseHandle(proc) //nolint:errcheck

	// Get the full executable path of the process
	filename := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(filename))

	if err = windows.QueryFullProcessImageName(proc, 0, &filename[0], &size); err != nil {
		return "", fmt.Errorf("query process name: %w", err)
	}

	// Convert it to a Go string and return
	return windows.UTF16ToString(filename), nil
}
