// mksyscall_windows.sh -l32 syscall_windows.go syscall_windows_386.go
// MACHINE GENERATED BY THE COMMAND ABOVE; DO NOT EDIT

package syscall

import "unsafe"

var (
	modkernel32 = loadDll("kernel32.dll")
	modwsock32  = loadDll("wsock32.dll")
	modws2_32   = loadDll("ws2_32.dll")

	procGetLastError               = getSysProcAddr(modkernel32, "GetLastError")
	procLoadLibraryW               = getSysProcAddr(modkernel32, "LoadLibraryW")
	procFreeLibrary                = getSysProcAddr(modkernel32, "FreeLibrary")
	procGetProcAddress             = getSysProcAddr(modkernel32, "GetProcAddress")
	procGetVersion                 = getSysProcAddr(modkernel32, "GetVersion")
	procFormatMessageW             = getSysProcAddr(modkernel32, "FormatMessageW")
	procExitProcess                = getSysProcAddr(modkernel32, "ExitProcess")
	procCreateFileW                = getSysProcAddr(modkernel32, "CreateFileW")
	procReadFile                   = getSysProcAddr(modkernel32, "ReadFile")
	procWriteFile                  = getSysProcAddr(modkernel32, "WriteFile")
	procSetFilePointer             = getSysProcAddr(modkernel32, "SetFilePointer")
	procCloseHandle                = getSysProcAddr(modkernel32, "CloseHandle")
	procGetStdHandle               = getSysProcAddr(modkernel32, "GetStdHandle")
	procFindFirstFileW             = getSysProcAddr(modkernel32, "FindFirstFileW")
	procFindNextFileW              = getSysProcAddr(modkernel32, "FindNextFileW")
	procFindClose                  = getSysProcAddr(modkernel32, "FindClose")
	procGetFileInformationByHandle = getSysProcAddr(modkernel32, "GetFileInformationByHandle")
	procGetCurrentDirectoryW       = getSysProcAddr(modkernel32, "GetCurrentDirectoryW")
	procSetCurrentDirectoryW       = getSysProcAddr(modkernel32, "SetCurrentDirectoryW")
	procCreateDirectoryW           = getSysProcAddr(modkernel32, "CreateDirectoryW")
	procRemoveDirectoryW           = getSysProcAddr(modkernel32, "RemoveDirectoryW")
	procDeleteFileW                = getSysProcAddr(modkernel32, "DeleteFileW")
	procMoveFileW                  = getSysProcAddr(modkernel32, "MoveFileW")
	procGetComputerNameW           = getSysProcAddr(modkernel32, "GetComputerNameW")
	procSetEndOfFile               = getSysProcAddr(modkernel32, "SetEndOfFile")
	procGetSystemTimeAsFileTime    = getSysProcAddr(modkernel32, "GetSystemTimeAsFileTime")
	procSleep                      = getSysProcAddr(modkernel32, "Sleep")
	procGetTimeZoneInformation     = getSysProcAddr(modkernel32, "GetTimeZoneInformation")
	procCreateIoCompletionPort     = getSysProcAddr(modkernel32, "CreateIoCompletionPort")
	procGetQueuedCompletionStatus  = getSysProcAddr(modkernel32, "GetQueuedCompletionStatus")
	procWSAStartup                 = getSysProcAddr(modwsock32, "WSAStartup")
	procWSACleanup                 = getSysProcAddr(modwsock32, "WSACleanup")
	procsocket                     = getSysProcAddr(modwsock32, "socket")
	procsetsockopt                 = getSysProcAddr(modwsock32, "setsockopt")
	procbind                       = getSysProcAddr(modwsock32, "bind")
	procconnect                    = getSysProcAddr(modwsock32, "connect")
	procgetsockname                = getSysProcAddr(modwsock32, "getsockname")
	procgetpeername                = getSysProcAddr(modwsock32, "getpeername")
	proclisten                     = getSysProcAddr(modwsock32, "listen")
	procshutdown                   = getSysProcAddr(modwsock32, "shutdown")
	procAcceptEx                   = getSysProcAddr(modwsock32, "AcceptEx")
	procGetAcceptExSockaddrs       = getSysProcAddr(modwsock32, "GetAcceptExSockaddrs")
	procWSARecv                    = getSysProcAddr(modws2_32, "WSARecv")
	procWSASend                    = getSysProcAddr(modws2_32, "WSASend")
)

func GetLastError() (lasterrno int) {
	r0, _, _ := Syscall(procGetLastError, 0, 0, 0)
	lasterrno = int(r0)
	return
}

func LoadLibrary(libname string) (handle uint32, errno int) {
	r0, _, e1 := Syscall(procLoadLibraryW, uintptr(unsafe.Pointer(StringToUTF16Ptr(libname))), 0, 0)
	handle = uint32(r0)
	if handle == 0 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func FreeLibrary(handle uint32) (ok bool, errno int) {
	r0, _, e1 := Syscall(procFreeLibrary, uintptr(handle), 0, 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func GetProcAddress(module uint32, procname string) (proc uint32, errno int) {
	r0, _, e1 := Syscall(procGetProcAddress, uintptr(module), uintptr(unsafe.Pointer(StringBytePtr(procname))), 0)
	proc = uint32(r0)
	if proc == 0 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func GetVersion() (ver uint32, errno int) {
	r0, _, e1 := Syscall(procGetVersion, 0, 0, 0)
	ver = uint32(r0)
	if ver == 0 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func FormatMessage(flags uint32, msgsrc uint32, msgid uint32, langid uint32, buf []uint16, args *byte) (n uint32, errno int) {
	var _p0 *uint16
	if len(buf) > 0 {
		_p0 = &buf[0]
	}
	r0, _, e1 := Syscall9(procFormatMessageW, uintptr(flags), uintptr(msgsrc), uintptr(msgid), uintptr(langid), uintptr(unsafe.Pointer(_p0)), uintptr(len(buf)), uintptr(unsafe.Pointer(args)), 0, 0)
	n = uint32(r0)
	if n == 0 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func ExitProcess(exitcode uint32) {
	Syscall(procExitProcess, uintptr(exitcode), 0, 0)
	return
}

func CreateFile(name *uint16, access uint32, mode uint32, sa *byte, createmode uint32, attrs uint32, templatefile int32) (handle int32, errno int) {
	r0, _, e1 := Syscall9(procCreateFileW, uintptr(unsafe.Pointer(name)), uintptr(access), uintptr(mode), uintptr(unsafe.Pointer(sa)), uintptr(createmode), uintptr(attrs), uintptr(templatefile), 0, 0)
	handle = int32(r0)
	if handle == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func ReadFile(handle int32, buf []byte, done *uint32, overlapped *Overlapped) (ok bool, errno int) {
	var _p0 *byte
	if len(buf) > 0 {
		_p0 = &buf[0]
	}
	r0, _, e1 := Syscall6(procReadFile, uintptr(handle), uintptr(unsafe.Pointer(_p0)), uintptr(len(buf)), uintptr(unsafe.Pointer(done)), uintptr(unsafe.Pointer(overlapped)), 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func WriteFile(handle int32, buf []byte, done *uint32, overlapped *Overlapped) (ok bool, errno int) {
	var _p0 *byte
	if len(buf) > 0 {
		_p0 = &buf[0]
	}
	r0, _, e1 := Syscall6(procWriteFile, uintptr(handle), uintptr(unsafe.Pointer(_p0)), uintptr(len(buf)), uintptr(unsafe.Pointer(done)), uintptr(unsafe.Pointer(overlapped)), 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func SetFilePointer(handle int32, lowoffset int32, highoffsetptr *int32, whence uint32) (newlowoffset uint32, errno int) {
	r0, _, e1 := Syscall6(procSetFilePointer, uintptr(handle), uintptr(lowoffset), uintptr(unsafe.Pointer(highoffsetptr)), uintptr(whence), 0, 0)
	newlowoffset = uint32(r0)
	if newlowoffset == 0xffffffff {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func CloseHandle(handle int32) (ok bool, errno int) {
	r0, _, e1 := Syscall(procCloseHandle, uintptr(handle), 0, 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func GetStdHandle(stdhandle int32) (handle int32, errno int) {
	r0, _, e1 := Syscall(procGetStdHandle, uintptr(stdhandle), 0, 0)
	handle = int32(r0)
	if handle == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func FindFirstFile(name *uint16, data *Win32finddata) (handle int32, errno int) {
	r0, _, e1 := Syscall(procFindFirstFileW, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(data)), 0)
	handle = int32(r0)
	if handle == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func FindNextFile(handle int32, data *Win32finddata) (ok bool, errno int) {
	r0, _, e1 := Syscall(procFindNextFileW, uintptr(handle), uintptr(unsafe.Pointer(data)), 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func FindClose(handle int32) (ok bool, errno int) {
	r0, _, e1 := Syscall(procFindClose, uintptr(handle), 0, 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func GetFileInformationByHandle(handle int32, data *ByHandleFileInformation) (ok bool, errno int) {
	r0, _, e1 := Syscall(procGetFileInformationByHandle, uintptr(handle), uintptr(unsafe.Pointer(data)), 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func GetCurrentDirectory(buflen uint32, buf *uint16) (n uint32, errno int) {
	r0, _, e1 := Syscall(procGetCurrentDirectoryW, uintptr(buflen), uintptr(unsafe.Pointer(buf)), 0)
	n = uint32(r0)
	if n == 0 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func SetCurrentDirectory(path *uint16) (ok bool, errno int) {
	r0, _, e1 := Syscall(procSetCurrentDirectoryW, uintptr(unsafe.Pointer(path)), 0, 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func CreateDirectory(path *uint16, sa *byte) (ok bool, errno int) {
	r0, _, e1 := Syscall(procCreateDirectoryW, uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(sa)), 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func RemoveDirectory(path *uint16) (ok bool, errno int) {
	r0, _, e1 := Syscall(procRemoveDirectoryW, uintptr(unsafe.Pointer(path)), 0, 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func DeleteFile(path *uint16) (ok bool, errno int) {
	r0, _, e1 := Syscall(procDeleteFileW, uintptr(unsafe.Pointer(path)), 0, 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func MoveFile(from *uint16, to *uint16) (ok bool, errno int) {
	r0, _, e1 := Syscall(procMoveFileW, uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(to)), 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func GetComputerName(buf *uint16, n *uint32) (ok bool, errno int) {
	r0, _, e1 := Syscall(procGetComputerNameW, uintptr(unsafe.Pointer(buf)), uintptr(unsafe.Pointer(n)), 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func SetEndOfFile(handle int32) (ok bool, errno int) {
	r0, _, e1 := Syscall(procSetEndOfFile, uintptr(handle), 0, 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func GetSystemTimeAsFileTime(time *Filetime) {
	Syscall(procGetSystemTimeAsFileTime, uintptr(unsafe.Pointer(time)), 0, 0)
	return
}

func sleep(msec uint32) {
	Syscall(procSleep, uintptr(msec), 0, 0)
	return
}

func GetTimeZoneInformation(tzi *Timezoneinformation) (rc uint32, errno int) {
	r0, _, e1 := Syscall(procGetTimeZoneInformation, uintptr(unsafe.Pointer(tzi)), 0, 0)
	rc = uint32(r0)
	if rc == 0xffffffff {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func CreateIoCompletionPort(filehandle int32, cphandle int32, key uint32, threadcnt uint32) (handle int32, errno int) {
	r0, _, e1 := Syscall6(procCreateIoCompletionPort, uintptr(filehandle), uintptr(cphandle), uintptr(key), uintptr(threadcnt), 0, 0)
	handle = int32(r0)
	if handle == 0 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func GetQueuedCompletionStatus(cphandle int32, qty *uint32, key *uint32, overlapped **Overlapped, timeout uint32) (ok bool, errno int) {
	r0, _, e1 := Syscall6(procGetQueuedCompletionStatus, uintptr(cphandle), uintptr(unsafe.Pointer(qty)), uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(overlapped)), uintptr(timeout), 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func WSAStartup(verreq uint32, data *WSAData) (sockerrno int) {
	r0, _, _ := Syscall(procWSAStartup, uintptr(verreq), uintptr(unsafe.Pointer(data)), 0)
	sockerrno = int(r0)
	return
}

func WSACleanup() (errno int) {
	r1, _, e1 := Syscall(procWSACleanup, 0, 0, 0)
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func socket(af int32, typ int32, protocol int32) (handle int32, errno int) {
	r0, _, e1 := Syscall(procsocket, uintptr(af), uintptr(typ), uintptr(protocol))
	handle = int32(r0)
	if handle == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func setsockopt(s int32, level int32, optname int32, optval *byte, optlen int32) (errno int) {
	r1, _, e1 := Syscall6(procsetsockopt, uintptr(s), uintptr(level), uintptr(optname), uintptr(unsafe.Pointer(optval)), uintptr(optlen), 0)
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func bind(s int32, name uintptr, namelen int32) (errno int) {
	r1, _, e1 := Syscall(procbind, uintptr(s), uintptr(name), uintptr(namelen))
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func connect(s int32, name uintptr, namelen int32) (errno int) {
	r1, _, e1 := Syscall(procconnect, uintptr(s), uintptr(name), uintptr(namelen))
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func getsockname(s int32, rsa *RawSockaddrAny, addrlen *int32) (errno int) {
	r1, _, e1 := Syscall(procgetsockname, uintptr(s), uintptr(unsafe.Pointer(rsa)), uintptr(unsafe.Pointer(addrlen)))
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func getpeername(s int32, rsa *RawSockaddrAny, addrlen *int32) (errno int) {
	r1, _, e1 := Syscall(procgetpeername, uintptr(s), uintptr(unsafe.Pointer(rsa)), uintptr(unsafe.Pointer(addrlen)))
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func listen(s int32, backlog int32) (errno int) {
	r1, _, e1 := Syscall(proclisten, uintptr(s), uintptr(backlog), 0)
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func shutdown(s int32, how int32) (errno int) {
	r1, _, e1 := Syscall(procshutdown, uintptr(s), uintptr(how), 0)
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func AcceptEx(ls uint32, as uint32, buf *byte, rxdatalen uint32, laddrlen uint32, raddrlen uint32, recvd *uint32, overlapped *Overlapped) (ok bool, errno int) {
	r0, _, e1 := Syscall9(procAcceptEx, uintptr(ls), uintptr(as), uintptr(unsafe.Pointer(buf)), uintptr(rxdatalen), uintptr(laddrlen), uintptr(raddrlen), uintptr(unsafe.Pointer(recvd)), uintptr(unsafe.Pointer(overlapped)), 0)
	ok = bool(r0 != 0)
	if !ok {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func GetAcceptExSockaddrs(buf *byte, rxdatalen uint32, laddrlen uint32, raddrlen uint32, lrsa **RawSockaddrAny, lrsalen *int32, rrsa **RawSockaddrAny, rrsalen *int32) {
	Syscall9(procGetAcceptExSockaddrs, uintptr(unsafe.Pointer(buf)), uintptr(rxdatalen), uintptr(laddrlen), uintptr(raddrlen), uintptr(unsafe.Pointer(lrsa)), uintptr(unsafe.Pointer(lrsalen)), uintptr(unsafe.Pointer(rrsa)), uintptr(unsafe.Pointer(rrsalen)), 0)
	return
}

func WSARecv(s uint32, bufs *WSABuf, bufcnt uint32, recvd *uint32, flags *uint32, overlapped *Overlapped, croutine *byte) (errno int) {
	r1, _, e1 := Syscall9(procWSARecv, uintptr(s), uintptr(unsafe.Pointer(bufs)), uintptr(bufcnt), uintptr(unsafe.Pointer(recvd)), uintptr(unsafe.Pointer(flags)), uintptr(unsafe.Pointer(overlapped)), uintptr(unsafe.Pointer(croutine)), 0, 0)
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}

func WSASend(s uint32, bufs *WSABuf, bufcnt uint32, sent *uint32, flags uint32, overlapped *Overlapped, croutine *byte) (errno int) {
	r1, _, e1 := Syscall9(procWSASend, uintptr(s), uintptr(unsafe.Pointer(bufs)), uintptr(bufcnt), uintptr(unsafe.Pointer(sent)), uintptr(flags), uintptr(unsafe.Pointer(overlapped)), uintptr(unsafe.Pointer(croutine)), 0, 0)
	if int(r1) == -1 {
		errno = int(e1)
	} else {
		errno = 0
	}
	return
}
