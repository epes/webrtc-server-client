package common

import "math/rand"

func RandString(length int) string {
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		bytes[i] = byte(65 + rand.Intn(26))
	}
	return string(bytes)
}
