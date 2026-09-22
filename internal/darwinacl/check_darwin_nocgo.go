//go:build darwin && !cgo

package darwinacl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	attrBitMapCount         = 5
	attrCommonExtendedACL   = 0x00400000
	attrCommonReturned      = 0x80000000
	attrVolumeCapabilities  = 0x00020000
	attrVolumeInfo          = 0x80000000
	attrReferenceOffset     = 24
	attrReferenceSize       = 8
	kauthFileSecuritySize   = 44
	kauthFileSecurityMagic  = 0x012cc16d
	kauthFileSecurityNoACL  = 0xffffffff
	extendedSecurityBufSize = 4096
	volumeCapabilitiesSize  = 32
	volumeCapabilityACL     = 0x00000400
)

var errPresent = errors.New("extended ACL present")

type attributeList struct {
	BitmapCount uint16
	Reserved    uint16
	Common      uint32
	Volume      uint32
	Directory   uint32
	File        uint32
	Fork        uint32
}

func CheckPrivate(file *os.File) error {
	if err := checkACLCapability(file); err != nil {
		return err
	}
	attributes := attributeList{
		BitmapCount: attrBitMapCount,
		Common:      attrCommonReturned | attrCommonExtendedACL,
	}
	var buffer [extendedSecurityBufSize]byte
	total, err := fgetattrlist(file, &attributes, buffer[:])
	if err != nil {
		return err
	}
	return classifyExtendedSecurity(buffer[:total])
}

func checkACLCapability(file *os.File) error {
	attributes := attributeList{
		BitmapCount: attrBitMapCount,
		Common:      attrCommonReturned,
		Volume:      attrVolumeInfo | attrVolumeCapabilities,
	}
	var buffer [attrReferenceOffset + volumeCapabilitiesSize]byte
	total, err := fgetattrlist(file, &attributes, buffer[:])
	if err != nil {
		return err
	}
	if total < len(buffer) {
		return errors.New("invalid ACL capability response")
	}
	returnedVolume := binary.NativeEndian.Uint32(buffer[8:12])
	if returnedVolume&attrVolumeCapabilities == 0 {
		return errors.New("ACL capability unavailable")
	}
	capabilities := buffer[attrReferenceOffset : attrReferenceOffset+volumeCapabilitiesSize]
	interfaces := binary.NativeEndian.Uint32(capabilities[4:8])
	validInterfaces := binary.NativeEndian.Uint32(capabilities[20:24])
	if validInterfaces&volumeCapabilityACL == 0 || interfaces&volumeCapabilityACL == 0 {
		return errors.New("extended ACL inspection unsupported")
	}
	return nil
}

func fgetattrlist(file *os.File, attributes *attributeList, buffer []byte) (int, error) {
	_, _, errno := unix.Syscall6(
		unix.SYS_FGETATTRLIST,
		file.Fd(),
		uintptr(unsafe.Pointer(attributes)),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
		0,
		0,
	)
	runtime.KeepAlive(file)
	if errno != 0 {
		return 0, fmt.Errorf("inspect extended ACL: %w", errno)
	}
	if len(buffer) < 4 {
		return 0, errors.New("invalid ACL response buffer")
	}
	total := int(binary.NativeEndian.Uint32(buffer[:4]))
	if total < 4 || total > len(buffer) {
		return 0, errors.New("invalid ACL response length")
	}
	return total, nil
}

func classifyExtendedSecurity(buffer []byte) error {
	if len(buffer) < attrReferenceOffset+attrReferenceSize {
		return errors.New("invalid extended ACL response")
	}
	returned := binary.NativeEndian.Uint32(buffer[4:8])
	if returned&attrCommonExtendedACL == 0 {
		return nil
	}
	dataOffset := int(int32(binary.NativeEndian.Uint32(buffer[attrReferenceOffset : attrReferenceOffset+4])))
	dataLength := int(binary.NativeEndian.Uint32(buffer[attrReferenceOffset+4 : attrReferenceOffset+8]))
	dataStart := attrReferenceOffset + dataOffset
	if dataOffset < attrReferenceSize || dataOffset%4 != 0 || dataLength < kauthFileSecuritySize || dataStart > len(buffer)-dataLength {
		return errors.New("invalid extended ACL payload")
	}
	security := buffer[dataStart : dataStart+dataLength]
	if binary.NativeEndian.Uint32(security[:4]) != kauthFileSecurityMagic {
		return errors.New("invalid extended ACL magic")
	}
	entryCount := binary.NativeEndian.Uint32(security[36:40])
	if entryCount != kauthFileSecurityNoACL {
		return errPresent
	}
	if dataLength != kauthFileSecuritySize {
		return errors.New("invalid empty extended ACL payload")
	}
	return nil
}
