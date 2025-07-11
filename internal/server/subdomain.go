package server

import (
	"math/rand/v2"
	"time"
)

func generateRandomCNAME(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	seed := uint64(time.Now().UnixNano()) // explicit conversion
	r := rand.New(rand.NewPCG(seed, 0))   // 0 is the stream ID
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.IntN(len(charset))]
	}
	return string(b)
}
