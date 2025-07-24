package utils

import (
	"encoding/binary"
	"crypto/rand"
	"math/big"
)

func RandFloat() float64 {

	buf := make([]byte, 8)
	rand.Read(buf)
	bits := binary.BigEndian.Uint32(buf)

	return float64(bits) / float64(1<<64)

}

func RandInt(ts int) int {

	n, _ := rand.Int(rand.Reader, big.NewInt(int64(ts)))

	return int(n.Int64())
}
