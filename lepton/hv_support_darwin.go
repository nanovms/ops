package lepton

import (
	"C"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func hvSupport() (bool, error) {
	name, err := unix.ByteSliceFromString("kern.hv_support")
	if err != nil {
		return false, err
	}

	// allocate the mib result buffer
	var buf [unix.CTL_MAXNAME + 2]C.int
	bufBytes := (*byte)(unsafe.Pointer(&buf[0]))
	sz := unsafe.Sizeof(buf[0])
	bufLen := uintptr(unix.CTL_MAXNAME) * sz

	// make system call to read in the mib
	if err = sysctl([]C.int{0, 3}, bufBytes, &bufLen, &name[0], uintptr(len(name))); err != nil {
		return false, err
	}
	mib := []C.int(buf[0 : bufLen/sz])

	// allocate the VT-X support result buffer
	var resBuf C.int
	resBufBytes := (*byte)(unsafe.Pointer(&resBuf))
	sz = unsafe.Sizeof(resBuf)

	// make the system call to read in the VT-X support result
	var nullPtr uintptr
	if err := sysctl(mib, resBufBytes, &sz, nil, nullPtr); err != nil {
		return false, err
	}

	return resBuf != 0, nil
}

func sysctl(mib []C.int, old *byte, oldlen *uintptr, new *byte, newlen uintptr) (err error) {
	var _zero uintptr
	var _p0 unsafe.Pointer
	if len(mib) > 0 {
		_p0 = unsafe.Pointer(&mib[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	_, _, e1 := unix.Syscall6(
		unix.SYS___SYSCTL,
		uintptr(_p0),
		uintptr(len(mib)),
		uintptr(unsafe.Pointer(old)),
		uintptr(unsafe.Pointer(oldlen)),
		uintptr(unsafe.Pointer(new)),
		uintptr(newlen))
	if e1 != 0 {
		return os.NewSyscallError("sysctl",
			fmt.Errorf("%d", e1))
	}
	return
}
