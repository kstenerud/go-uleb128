package uleb128

import (
	"fmt"
	"math/big"
)

// https://en.wikipedia.org/wiki/LEB128

// EncodedSize returns the number of bytes required to encode this value
func EncodedSize(value *big.Int) int {
	if isZero(value) {
		return 1
	}
	words := value.Bits()
	bits := (len(words) - 1) * wordSize()
	groupCount := bits/7 + 1
	bitOffset := 7 - bits%7
	highWord := words[len(words)-1] >> bitOffset
	for highWord != 0 {
		groupCount++
		highWord >>= 7
	}
	return groupCount
}

func Encode(value *big.Int, buffer []byte) {
	if is32Bit() {
		encode32(value, buffer)
	}
	encode64(value, buffer)
}

func Decode(buffer []byte) (large *big.Int, byteCount int, err error) {
	words := []big.Word{}
	word := big.Word(0)
	bitIndex := 0
	for index := 0; index < len(buffer); index++ {
		b := buffer[index]
		word |= big.Word(b&payloadMask) << bitIndex

		bitIndex += 7
		if bitIndex >= wordSize() {
			words = append(words, big.Word(word))
			bitIndex &= int(wordMask())
			word = big.Word(b&payloadMask) >> (7 - bitIndex)
		}

		if b&continuationMask != continuationMask {
			if word != 0 {
				words = append(words, big.Word(word))
			}
			large = big.NewInt(0)
			large.SetBits(words)
			byteCount = index + 1
			return
		}
	}

	err = fmt.Errorf("Unterminated uleb128 value")

	return
}

func encode32(value *big.Int, buffer []byte) {
	if !is32Bit() {
		return
	}

	if isZero(value) {
		buffer[0] = 0
		return
	}

	const lowMask = 0xffff
	const highMask = 0xffff0000
	words := value.Bits()
	accum := big.Word(0)
	bufferPos := 0
	end := len(words) - 1
	shiftIndex := 0
	shift := 0

	for i := 0; i < end; i++ {
		srcWord := words[i]

		// Low 16 bits
		shift = int(leftShifts32[shiftIndex])
		accum |= (srcWord & lowMask) << shift
		groupCount := int(groupCounts32[shiftIndex])
		for j := 0; j < groupCount; j++ {
			buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
			accum >>= 7
			bufferPos++
		}

		shiftIndex = (shiftIndex + 1) % 15

		// High 16 bits
		shift = int(rightShifts32[shiftIndex])
		accum |= (srcWord & highMask) >> shift
		groupCount = int(groupCounts32[shiftIndex])
		for j := 0; j < groupCount; j++ {
			buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
			accum >>= 7
			bufferPos++
		}

		shiftIndex = (shiftIndex + 1) % 15
	}

	srcWord := words[end]

	// Low 16 bits
	shift = int(leftShifts32[shiftIndex])
	srcWordHigh := srcWord & highMask
	accum |= (srcWord & lowMask) << shift
	groupCount := int(groupCounts32[shiftIndex])
	for j := 0; j < groupCount; j++ {
		buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
		accum >>= 7
		if accum == 0 && srcWordHigh == 0 {
			buffer[bufferPos] &= payloadMask
			return
		}
		bufferPos++
	}

	shiftIndex = (shiftIndex + 1) % 15

	// High 16 bits
	shift = int(rightShifts32[shiftIndex])
	accum |= (srcWord & highMask) >> shift
	groupCount = int(groupCounts32[shiftIndex])
	for {
		buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
		accum >>= 7
		if accum == 0 {
			buffer[bufferPos] &= payloadMask
			return
		}
		bufferPos++
	}
}

func encode64(value *big.Int, buffer []byte) {
	if is32Bit() {
		return
	}

	if isZero(value) {
		buffer[0] = 0
		return
	}

	const lowMask = 0xffffffff
	const highMask = 0xffffffff << 32
	words := value.Bits()
	accum := big.Word(0)
	bufferPos := 0
	end := len(words) - 1
	shiftIndex := 0
	shift := 0

	for i := 0; i < end; i++ {
		srcWord := words[i]

		// Low 32 bits
		shift = shiftIndex / 2
		accum |= (srcWord & lowMask) << shift
		groupCount := int(groupCounts64[shiftIndex])
		for j := 0; j < groupCount; j++ {
			buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
			accum >>= 7
			bufferPos++
		}

		shiftIndex = (shiftIndex + 1) % 15

		// High 32 bits
		shift = int(rightShifts64[shiftIndex])
		accum |= (srcWord & highMask) >> shift
		groupCount = int(groupCounts64[shiftIndex])
		for j := 0; j < groupCount; j++ {
			buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
			accum >>= 7
			bufferPos++
		}

		shiftIndex = (shiftIndex + 1) % 15
	}

	srcWord := words[end]

	// Low 32 bits
	shift = shiftIndex / 2
	srcWordHigh := srcWord & highMask
	accum |= (srcWord & lowMask) << shift
	groupCount := int(groupCounts64[shiftIndex])
	for j := 0; j < groupCount; j++ {
		buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
		accum >>= 7
		if accum == 0 && srcWordHigh == 0 {
			buffer[bufferPos] &= payloadMask
			return
		}
		bufferPos++
	}

	shiftIndex = (shiftIndex + 1) % 15

	// High 32 bits
	shift = int(rightShifts64[shiftIndex])
	accum |= srcWordHigh >> shift
	groupCount = int(groupCounts64[shiftIndex])
	for {
		buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
		accum >>= 7
		if accum == 0 {
			buffer[bufferPos] &= payloadMask
			return
		}
		bufferPos++
	}
}

func is32Bit() bool {
	return ^uint(0) == 0xffffffff
}

func wordSize() int {
	if is32Bit() {
		return 32
	}
	return 64
}

func wordMask() uint {
	return uint(wordSize()) - 1
}

func isZero(v *big.Int) bool {
	return v.BitLen() == 0
}

const payloadMask = 0x7f
const continuationMask = 0x80

// 64-bit
// | Step | U/L | shift | groups | remain |
// | ---- | --- | ----- | ------ | ------ |
// |    0 |  L  |    0  |    4   |    4   |
// |    1 |  H  |  -28  |    5   |    1   |
// |    2 |  L  |    1  |    4   |    5   |
// |    3 |  H  |  -27  |    5   |    2   |
// |    4 |  L  |    2  |    4   |    6   |
// |    5 |  H  |  -26  |    5   |    3   |
// |    6 |  L  |    3  |    5   |    0   |
// |    7 |  H  |  -32  |    4   |    4   |
// |    8 |  L  |    4  |    5   |    1   |
// |    9 |  H  |  -31  |    4   |    5   |
// |   10 |  L  |    5  |    5   |    2   |
// |   11 |  H  |  -30  |    4   |    6   |
// |   12 |  L  |    6  |    5   |    3   |
// |   13 |  H  |  -29  |    5   |    0   |
var groupCounts64 = []uint8{4, 5, 4, 5, 4, 5, 5, 4, 5, 4, 5, 4, 5, 5}
var rightShifts64 = []uint8{0, 28, 0, 27, 0, 26, 0, 32, 0, 31, 0, 30, 0, 29}

// 32-bit
// | Step | U/L | shift | groups | remain |
// | ---- | --- | ----- | ------ | ------ |
// |    0 |  L  |    0  |    2   |    2   |
// |    1 |  H  |  -14  |    2   |    4   |
// |    2 |  L  |    4  |    2   |    6   |
// |    3 |  H  |  -10  |    3   |    1   |
// |    4 |  L  |    1  |    2   |    3   |
// |    5 |  H  |  -13  |    2   |    5   |
// |    6 |  L  |    5  |    3   |    0   |
// |    7 |  H  |  -16  |    2   |    2   |
// |    8 |  L  |    2  |    2   |    4   |
// |    9 |  H  |  -12  |    2   |    6   |
// |   10 |  L  |    6  |    3   |    1   |
// |   11 |  H  |  -15  |    2   |    3   |
// |   12 |  L  |    3  |    2   |    5   |
// |   13 |  H  |  -11  |    3   |    0   |
var groupCounts32 = []uint8{2, 2, 2, 3, 2, 2, 3, 2, 2, 2, 3, 2, 2, 3}
var leftShifts32 = []uint8{0, 0, 4, 0, 1, 0, 5, 0, 2, 0, 6, 0, 3, 0}
var rightShifts32 = []uint8{0, 14, 0, 10, 0, 13, 0, 16, 0, 12, 0, 15, 0, 11}
