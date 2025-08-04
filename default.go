package shardedflight

import (
	"unsafe"
)

// // // // // // // //

const (
	offset64 = uint64(14695981039346656037)
	prime64  = uint64(1099511628211)
)

func unsafeString(b []byte) string { return *(*string)(unsafe.Pointer(&b)) }

// //

// defaultBuilder the most cheap concalation without allocation
func defaultBuilder(parts ...string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		total := 0
		for _, p := range parts {
			total += len(p)
		}
		b := make([]byte, 0, total)
		for _, p := range parts {
			b = append(b, p...)
		}
		return unsafeString(b)
	}
}

// defaultHash  64-bit FNV-1a; On average ~ 1ns per key
func defaultHash(s string) uint64 {
	h := offset64
	bs := unsafe.Slice(unsafe.StringData(s), len(s))

	for _, b := range bs {
		h ^= uint64(b)
		h *= prime64
	}
	return h
}
