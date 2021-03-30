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
	"io"
	"math/big"
)

// Maximum number of bytes that will ever be written to a buffer
const MaxBufferWriteBytes = 10

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

// Encode a math.big.Int value (the sign of the value will be ignored).
func Encode(value *big.Int, writer io.Writer) (byteCount int, err error) {
	buffer := make([]byte, EncodedSize(value))
	byteCount = EncodeToBytes(value, buffer)
	return writer.Write(buffer[:byteCount])
}

// Encode a math.big.Int value (the sign of the value will be ignored).
// Assumes that there's enough room in buffer (see MaxBufferWriteBytes).
func EncodeToBytes(value *big.Int, buffer []byte) (byteCount int) {
	// Manually dispatch based on architecture because +build can't select on
	// word size. This is awkward, but should in theory be optimized away.
	if is32Bit() {
		return encode32(value, buffer)
	}
	return encode64(value, buffer)
}

// Encode a uint64 value, returning the number of bytes encoded.
func EncodeUint64(value uint64, writer io.Writer) (byteCount int, err error) {
	buffer := make([]byte, 10)
	byteCount = EncodeUint64ToBytes(value, buffer)
	return writer.Write(buffer[:byteCount])
}

// Encode a uint64 value, returning the number of bytes encoded.
// Assumes that there's enough room in buffer (see MaxBufferWriteBytes).
func EncodeUint64ToBytes(value uint64, buffer []byte) (byteCount int) {
	const lastByteMask = ^uint64(0x7f)

	if (value & lastByteMask) == 0 {
		buffer[0] = byte(value)
		byteCount = 1
		return
	}

	continueFlag := uint64(continuationMask)
	for value != 0 {
		buffer[byteCount] = byte((value & payloadMask) | continueFlag)
		byteCount++
		value >>= 7
		if (value & lastByteMask) == 0 {
			continueFlag = 0
		}
	}
	return
}

// Decode a ULEB128 value.
// If the result is small enough to fit into type uint64, asBigInt will be nil
// and asUint will contain the result.
func Decode(reader io.Reader) (asUint uint64, asBigInt *big.Int, byteCount int, err error) {
	buffer := []byte{0}
	return DecodeWithByteBuffer(reader, buffer)
}

// Decode a ULEB128 value using the supplied 1-byte buffer (to avoid extra allocations).
// If the result is small enough to fit into type uint64, asBigInt will be nil
// and asUint will contain the result.
func DecodeWithByteBuffer(reader io.Reader, buffer []byte) (asUint uint64, asBigInt *big.Int, byteCount int, err error) {
	buffer = buffer[:1]
	if _, err = reader.Read(buffer); err != nil {
		return
	}
	byteCount = 1
	if buffer[0] < 0x80 {
		asUint = uint64(buffer[0])
		return
	}

	words := []big.Word{}

	word := big.Word(buffer[0] & payloadMask)
	bitIndex := uint(7)
	bytesRead := 0
	for {
		bytesRead, err = reader.Read(buffer[:])
		if bytesRead == 0 {
			return
		}
		byteCount++
		word |= big.Word(buffer[0]&payloadMask) << bitIndex

		bitIndex += 7
		if int(bitIndex) >= wordSize() {
			words = append(words, big.Word(word))
			bitIndex &= wordMask()
			word = big.Word(buffer[0]&payloadMask) >> (7 - bitIndex)
		}

		if buffer[0]&continuationMask != continuationMask {
			if len(words) == 0 {
				asUint = uint64(word)
				return
			}
			if word != 0 {
				words = append(words, big.Word(word))
			}
			if is32Bit() {
				if len(words) == 1 {
					asUint = uint64(words[0])
					return
				} else if len(words) == 2 {
					asUint = (uint64(words[1]) << 32) | uint64(words[0])
					return
				}
			} else {
				if len(words) == 1 {
					asUint = uint64(words[0])
					return
				}
			}
			asBigInt = big.NewInt(0)
			asBigInt.SetBits(words)
			return
		}
	}

	return
}

func maskForBitCount(bitCount int) uint64 {
	return ^(^uint64(0) << uint(bitCount))
}

func encode32(value *big.Int, buffer []byte) (byteCount int) {
	// Prevent compilation on 64-bit arch
	if !is32Bit() {
		return
	}

	if isZero(value) {
		buffer[0] = 0
		byteCount = 1
		return
	}

	const lowMask = 0xffff
	const highMask = 0xffff0000
	words := value.Bits()
	accum := big.Word(0)
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
			buffer[byteCount] = byte(accum&payloadMask) | continuationMask
			byteCount++
			accum >>= 7
		}

		shiftIndex = (shiftIndex + 1) % 15

		// High 16 bits
		shift = uint(rightShifts32[shiftIndex])
		accum |= (srcWord & highMask) >> shift
		groupCount = int(groupCounts32[shiftIndex])
		for j := 0; j < groupCount; j++ {
			buffer[byteCount] = byte(accum&payloadMask) | continuationMask
			byteCount++
			accum >>= 7
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
		buffer[byteCount] = byte(accum & payloadMask)
		byteCount++
		accum >>= 7
		if accum == 0 && srcWordHigh == 0 {
			return
		}
		buffer[byteCount-1] |= continuationMask
	}

	shiftIndex = (shiftIndex + 1) % 15

	// High 16 bits
	shift = uint(rightShifts32[shiftIndex])
	accum |= (srcWord & highMask) >> shift
	groupCount = int(groupCounts32[shiftIndex])
	for {
		buffer[byteCount] = byte(accum & payloadMask)
		byteCount++
		accum >>= 7
		if accum == 0 {
			return
		}
		buffer[byteCount-1] |= continuationMask
	}
	return
}

func encode64(value *big.Int, buffer []byte) (byteCount int) {
	// Prevent compilation on 32-bit arch
	if is32Bit() {
		return
	}

	if isZero(value) {
		buffer[0] = 0
		byteCount = 1
		return
	}

	const lowMask = 0xffffffff
	highMask := big.Word(0xffffffff)
	highMask <<= 32
	words := value.Bits()
	accum := big.Word(0)
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
			buffer[byteCount] = byte(accum&payloadMask) | continuationMask
			byteCount++
			accum >>= 7
		}

		shiftIndex = (shiftIndex + 1) % 15

		// High 32 bits
		shift = uint(rightShifts64[shiftIndex])
		accum |= (srcWord & highMask) >> shift
		groupCount = int(groupCounts64[shiftIndex])
		for j := 0; j < groupCount; j++ {
			buffer[byteCount] = byte(accum&payloadMask) | continuationMask
			byteCount++
			accum >>= 7
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
		buffer[byteCount] = byte(accum & payloadMask)
		byteCount++
		accum >>= 7
		if accum == 0 && srcWordHigh == 0 {
			return
		}
		buffer[byteCount-1] |= continuationMask
	}

	shiftIndex = (shiftIndex + 1) % 15

	// High 32 bits
	shift = uint(rightShifts64[shiftIndex])
	accum |= srcWordHigh >> shift
	groupCount = int(groupCounts64[shiftIndex])
	for {
		buffer[byteCount] = byte(accum & payloadMask)
		byteCount++
		accum >>= 7
		if accum == 0 {
			return
		}
		buffer[byteCount-1] |= continuationMask
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
