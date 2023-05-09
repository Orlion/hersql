package util

import (
	"math/rand"
	"time"
)

func RandomBuf(size int) []byte {
	buf := make([]byte, size)
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < size; i++ {
		buf[i] = byte(rand.Intn(127))
		if buf[i] == 0 || buf[i] == byte('$') {
			buf[i]++
		}
	}
	return buf
}
