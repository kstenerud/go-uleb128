// Copyright 2020 Karl Stenerud
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
// IN THE SOFTWARE.

// Package uleb128 provides encoders and decoders for unsigned little endian
// base128 values: https://en.wikipedia.org/wiki/LEB128
package uleb128

import (
	"math/big"
)

// EncodedSize returns the number of bytes required to encode this value.
func EncodedSize(value *big.Int) int {
	if isZero(value) {
		return 1
	}
	words := value.Bits()
	bits := (len(words) - 1) * wordSize()
	groupCount := bits/7 + 1
	bitOffset := uint(7 - bits%7)
	highWord := words[len(words)-1] >> bitOffset
	for highWord != 0 {
		groupCount++
		highWord >>= 7
	}
	return groupCount
}

// EncodedSizeUint64 returns the number of bytes required to encode this value.
func EncodedSizeUint64(value uint64) int {
	if value == 0 {
		return 1
	}
	groupCount := 0
	for value != 0 {
		groupCount++
		value >>= 7
	}
	return groupCount
}

// Encode a math.big.Int value, returning the number of bytes encoded. The sign
// of the value will be ignored.
// ok will be false if there wasn't enough room.
func Encode(value *big.Int, buffer []byte) (byteCount int, ok bool) {
	// Manually dispatch based on architecture because +build can't select on
	// word size. This is awkward, but should in theory be optimized away.
	if is32Bit() {
		return encode32(value, buffer)
	}
	return encode64(value, buffer)
}

// Encode a uint64 value, returning the number of bytes encoded.
// ok will be false if there wasn't enough room.
func EncodeUint64(value uint64, buffer []byte) (byteCount int, ok bool) {
	if value == 0 {
		if len(buffer) == 0 {
			return
		}
		buffer[0] = 0
		byteCount = 1
		ok = true
		return
	}

	index := 0
	for value != 0 {
		if index >= len(buffer) {
			return
		}
		buffer[index] = byte((value & payloadMask) | continuationMask)
		value >>= 7
		index++
	}
	buffer[index-1] &= payloadMask
	byteCount = index
	ok = true
	return
}

// Decode an encoded ULEB128 value. If the result is small enough to fit into
// type uint64, asBigInt will be nil and asUint will contain the result.
//
// preValue and preBitCount set the initial low bits of the decoded value, upon
// which decoded data is added, to support multipart operations (where a
// previous operation provides part of the first decoded word). Set to 0 to
// disable this. preValue will be masked according to preBitCount.
//
// ok will be false if the buffer wasn't terminated with a byte value < 0x80.
// ok will also be false if buffer is empty (in which case asUint will contain
// the masked preValue).
func Decode(preValue uint64, preBitCount int, buffer []byte) (asUint uint64, asBigInt *big.Int, byteCount int, ok bool) {
	if preBitCount > 64 {
		preBitCount = 64
	}
	preValue &= maskForBitCount(preBitCount)

	if len(buffer) == 0 {
		asUint = preValue
		return
	}

	if buffer[0] == 0 {
		byteCount = 1
		asUint = preValue
		ok = true
		return
	}

	if buffer[0] < 0x80 && preBitCount <= 57 {
		asUint = (uint64(buffer[0]) << uint(preBitCount)) | preValue
		byteCount = 1
		ok = true
		return
	}

	words := []big.Word{}

	for preBitCount >= wordSize() {
		words = append(words, big.Word(preValue))
		preValue >>= uint(wordSize())
		preBitCount -= wordSize()
	}

	word := big.Word(preValue)
	bitIndex := uint(preBitCount)
	for index := 0; index < len(buffer); index++ {
		b := buffer[index]
		word |= big.Word(b&payloadMask) << bitIndex

		bitIndex += 7
		if int(bitIndex) >= wordSize() {
			words = append(words, big.Word(word))
			bitIndex &= wordMask()
			word = big.Word(b&payloadMask) >> (7 - bitIndex)
		}

		if b&continuationMask != continuationMask {
			byteCount = index + 1
			if len(words) == 0 {
				asUint = uint64(word)
				ok = true
				return
			}
			if word != 0 {
				words = append(words, big.Word(word))
			}
			if is32Bit() {
				if len(words) == 1 {
					asUint = uint64(words[0])
					ok = true
					return
				} else if len(words) == 2 {
					asUint = (uint64(words[1]) << 32) | uint64(words[0])
					ok = true
					return
				}
			} else {
				if len(words) == 1 {
					asUint = uint64(words[0])
					ok = true
					return
				}
			}
			asBigInt = big.NewInt(0)
			asBigInt.SetBits(words)
			ok = true
			return
		}
	}

	return
}

func maskForBitCount(bitCount int) uint64 {
	return ^(^uint64(0) << uint(bitCount))
}

func encode32(value *big.Int, buffer []byte) (byteCount int, ok bool) {
	// Prevent compilation on 64-bit arch
	if !is32Bit() {
		return
	}

	if isZero(value) {
		if len(buffer) == 0 {
			return
		}
		buffer[0] = 0
		byteCount = 1
		ok = true
		return
	}

	const lowMask = 0xffff
	const highMask = 0xffff0000
	words := value.Bits()
	accum := big.Word(0)
	bufferPos := 0
	end := len(words) - 1
	shiftIndex := 0
	shift := uint(0)

	for i := 0; i < end; i++ {
		srcWord := words[i]

		// Low 16 bits
		shift = uint(leftShifts32[shiftIndex])
		accum |= (srcWord & lowMask) << shift
		groupCount := int(groupCounts32[shiftIndex])
		for j := 0; j < groupCount; j++ {
			if bufferPos >= len(buffer) {
				return
			}
			buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
			accum >>= 7
			bufferPos++
		}

		shiftIndex = (shiftIndex + 1) % 15

		// High 16 bits
		shift = uint(rightShifts32[shiftIndex])
		accum |= (srcWord & highMask) >> shift
		groupCount = int(groupCounts32[shiftIndex])
		for j := 0; j < groupCount; j++ {
			if bufferPos >= len(buffer) {
				return
			}
			buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
			accum >>= 7
			bufferPos++
		}

		shiftIndex = (shiftIndex + 1) % 15
	}

	srcWord := words[end]

	// Low 16 bits
	shift = uint(leftShifts32[shiftIndex])
	srcWordHigh := srcWord & highMask
	accum |= (srcWord & lowMask) << shift
	groupCount := int(groupCounts32[shiftIndex])
	for j := 0; j < groupCount; j++ {
		if bufferPos >= len(buffer) {
			return
		}
		buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
		accum >>= 7
		if accum == 0 && srcWordHigh == 0 {
			buffer[bufferPos] &= payloadMask
			byteCount = bufferPos + 1
			ok = true
			return
		}
		bufferPos++
	}

	shiftIndex = (shiftIndex + 1) % 15

	// High 16 bits
	shift = uint(rightShifts32[shiftIndex])
	accum |= (srcWord & highMask) >> shift
	groupCount = int(groupCounts32[shiftIndex])
	for {
		if bufferPos >= len(buffer) {
			return
		}
		buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
		accum >>= 7
		if accum == 0 {
			buffer[bufferPos] &= payloadMask
			byteCount = bufferPos + 1
			ok = true
			return
		}
		bufferPos++
	}
	byteCount = bufferPos
	ok = true
	return
}

func encode64(value *big.Int, buffer []byte) (byteCount int, ok bool) {
	// Prevent compilation on 32-bit arch
	if is32Bit() {
		return
	}

	if isZero(value) {
		if len(buffer) == 0 {
			return
		}
		buffer[0] = 0
		byteCount = 1
		ok = true
		return
	}

	const lowMask = 0xffffffff
	highMask := big.Word(0xffffffff)
	highMask <<= 32
	words := value.Bits()
	accum := big.Word(0)
	bufferPos := 0
	end := len(words) - 1
	shiftIndex := uint(0)
	shift := uint(0)

	for i := 0; i < end; i++ {
		srcWord := words[i]

		// Low 32 bits
		shift = shiftIndex / 2
		accum |= (srcWord & lowMask) << shift
		groupCount := int(groupCounts64[shiftIndex])
		for j := 0; j < groupCount; j++ {
			if bufferPos >= len(buffer) {
				return
			}
			buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
			accum >>= 7
			bufferPos++
		}

		shiftIndex = (shiftIndex + 1) % 15

		// High 32 bits
		shift = uint(rightShifts64[shiftIndex])
		accum |= (srcWord & highMask) >> shift
		groupCount = int(groupCounts64[shiftIndex])
		for j := 0; j < groupCount; j++ {
			if bufferPos >= len(buffer) {
				return
			}
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
		if bufferPos >= len(buffer) {
			return
		}
		buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
		accum >>= 7
		if accum == 0 && srcWordHigh == 0 {
			buffer[bufferPos] &= payloadMask
			byteCount = bufferPos + 1
			ok = true
			return
		}
		bufferPos++
	}

	shiftIndex = (shiftIndex + 1) % 15

	// High 32 bits
	shift = uint(rightShifts64[shiftIndex])
	accum |= srcWordHigh >> shift
	groupCount = int(groupCounts64[shiftIndex])
	for {
		if bufferPos >= len(buffer) {
			return
		}
		buffer[bufferPos] = byte(accum&payloadMask) | continuationMask
		accum >>= 7
		if accum == 0 {
			buffer[bufferPos] &= payloadMask
			byteCount = bufferPos + 1
			ok = true
			return
		}
		bufferPos++
	}
	byteCount = bufferPos
	ok = true
	return
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

// 64-bit words, split into upper and lower 32-bit groups:
//
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

// 32-bit words, split into upper and lower 16-bit groups:
//
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
