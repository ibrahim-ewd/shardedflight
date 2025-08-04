package shardedflight

import (
	"hash/fnv"
	"unsafe"
)

// // // // // // // //

func unsafeString(b []byte) string { return *(*string)(unsafe.Pointer(&b)) }

func unsafeStringBytes(s string) []byte {
	sh := (*[2]uintptr)(unsafe.Pointer(&s))
	return unsafe.Slice((*byte)(unsafe.Pointer(sh[0])), sh[1])
}

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
	h := fnv.New64a()
	_, _ = h.Write(unsafeStringBytes(s))
	return h.Sum64()
}
