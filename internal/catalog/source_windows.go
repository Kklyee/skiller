package catalog

import (
	"encoding/binary"
	"os"
	"syscall"
)

const reparseTagMountPoint = 0xa0000003

func linkSource(path string) (Source, string, bool) {
	target, err := os.Readlink(path)
	if err != nil {
		return SourceDirectory, "", false
	}

	if isJunction(path) {
		return SourceJunction, target, true
	}

	return SourceSymlink, target, true
}

func isJunction(path string) bool {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}

	handle, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)

	buffer := make([]byte, syscall.MAXIMUM_REPARSE_DATA_BUFFER_SIZE)
	var bytesReturned uint32
	if err := syscall.DeviceIoControl(
		handle,
		syscall.FSCTL_GET_REPARSE_POINT,
		nil,
		0,
		&buffer[0],
		uint32(len(buffer)),
		&bytesReturned,
		nil,
	); err != nil || bytesReturned < 4 {
		return false
	}

	return binary.LittleEndian.Uint32(buffer[:4]) == reparseTagMountPoint
}
