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

package uleb128

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/kstenerud/go-describe"
)

func toBigWords(words []uint64) (result []big.Word) {
	if is32Bit() {
		return toBigWords32(words)
	}
	return toBigWords64(words)
}

func toBigWords64(words []uint64) (result []big.Word) {
	for _, word := range words {
		result = append(result, big.Word(word))
	}
	return
}

func toBigWords32(words []uint64) (result []big.Word) {
	for _, word := range words {
		result = append(result, big.Word(word&0xffffffff))
		result = append(result, big.Word((word>>32)&0xffffffff))
	}
	return
}

func assertEncodeDecode(t *testing.T, words []uint64, expectedBytes ...byte) {
	expectedBigInt := big.NewInt(0)
	expectedBigInt.SetBits(toBigWords(words))
	expectedByteCount := EncodedSize(expectedBigInt)
	actualBuffer := &bytes.Buffer{}
	actualFilledByteCount, err := Encode(expectedBigInt, actualBuffer)
	if err != nil {
		t.Error(err)
		return
	}
	if actualFilledByteCount != expectedByteCount {
		t.Errorf("Expected %v to encode to a byte count of %v but got %v", describe.D(words), expectedByteCount, actualFilledByteCount)
		return
	}
	if actualFilledByteCount != actualBuffer.Len() {
		t.Errorf("Encoding %v reported a byte count of %v but was actually %v", describe.D(words), actualFilledByteCount, actualBuffer.Len())
		return
	}
	if !reflect.DeepEqual(actualBuffer.Bytes(), expectedBytes) {
		t.Errorf("Expected %v to encode to %v but got %v", describe.D(words), describe.D(expectedBytes), describe.D(actualBuffer.Bytes()))
		return
	}
	actualUint, actualBigInt, actualByteCount, err := Decode(bytes.NewBuffer(expectedBytes))
	if err != nil {
		t.Error(err)
		return
	}
	if actualByteCount != expectedByteCount {
		t.Errorf("Decoding %v: Expected decoding %v to have a byte count of %v but got %v", describe.D(words), describe.D(expectedBytes), expectedByteCount, actualByteCount)
		return
	}
	if actualBigInt != nil {
		if expectedBigInt.Cmp(actualBigInt) != 0 {
			t.Errorf("Expected %v to decode to big %x but got %x", describe.D(expectedBytes), expectedBigInt, actualBigInt)
			return
		}
	} else {
		if expectedBigInt.Uint64() != actualUint {
			t.Errorf("Expected %v to decode to %x but got %x", describe.D(expectedBytes), expectedBigInt.Uint64(), actualUint)
			return
		}
	}

	if len(words) > 1 {
		return
	}

	expectedUint := words[0]
	expectedByteCount = EncodedSizeUint64(expectedUint)
	actualBuffer.Reset()

	actualByteCount, err = EncodeUint64(expectedUint, actualBuffer)
	if err != nil {
		t.Error(err)
		return
	}
	if actualBuffer.Len() != expectedByteCount {
		t.Errorf("Encode 2: Expected byte count of %v but got %v", expectedByteCount, actualBuffer.Len())
		return
	}

	if !reflect.DeepEqual(actualBuffer.Bytes(), expectedBytes) {
		t.Errorf("Encode 2: Expected %v but got %v", describe.D(expectedBytes), describe.D(actualBuffer.Bytes()))
		return
	}
	actualUint, actualBigInt, actualByteCount, err = Decode(actualBuffer)
	if err != nil {
		t.Error(err)
		return
	}
	if actualByteCount != expectedByteCount {
		t.Errorf("Decode 2: Expected byte count of %v but got %v", expectedByteCount, actualByteCount)
		return
	}
	if expectedUint != actualUint {
		t.Errorf("Decode 2: Expected %x but got %x", expectedBigInt.Uint64(), actualUint)
		return
	}
}

func assertEncodeDecodeUint(t *testing.T, value uint64, expectedBytes ...byte) {
	expectedByteCount := EncodedSizeUint64(value)
	actualBuffer := &bytes.Buffer{}
	actualFilledByteCount, err := EncodeUint64(value, actualBuffer)
	if err != nil {
		t.Error(err)
		return
	}
	if actualFilledByteCount != expectedByteCount {
		t.Errorf("Expected %v to encode to a byte count of %v but got %v", value, expectedByteCount, actualFilledByteCount)
		return
	}
	if !reflect.DeepEqual(actualBuffer.Bytes(), expectedBytes) {
		t.Errorf("Expected %v to encode to %v but got %v", value, describe.D(expectedBytes), describe.D(actualBuffer.Bytes()))
		return
	}

	actualUint, actualBigInt, actualByteCount, err := Decode(bytes.NewBuffer(expectedBytes))
	if err != nil {
		t.Error(err)
		return
	}
	if actualByteCount != expectedByteCount {
		t.Errorf("Decoding %v: Expected decoding %v to have a byte count of %v but got %v", value, describe.D(expectedBytes), expectedByteCount, actualByteCount)
		return
	}
	if actualBigInt != nil {
		t.Errorf("%v (from %v) should not decode to a big int", describe.D(expectedBytes), value)
		return
	}
	if value != actualUint {
		t.Errorf("Expected %v to decode to 0x%x but got 0x%x", describe.D(expectedBytes), value, actualUint)
		return
	}
}

func assertDecode(t *testing.T, expectedWords []uint64, b ...byte) {
	expectedBigInt := big.NewInt(0)
	expectedBigInt.SetBits(toBigWords(expectedWords))
	expectedByteCount := len(b)
	buff := bytes.NewBuffer(b)

	actualUint, actualBigInt, actualByteCount, err := Decode(buff)
	if err != nil {
		t.Error(err)
		return
	}
	if actualByteCount != expectedByteCount {
		t.Errorf("Decode 1: Expected byte count of %v but got %v", expectedByteCount, actualByteCount)
		return
	}
	if actualBigInt != nil {
		if expectedBigInt.Cmp(actualBigInt) != 0 {
			t.Errorf("Decode 1 (big): Expected %x but got %x", expectedBigInt, actualBigInt)
			return
		}
	} else {
		if expectedBigInt.Uint64() != actualUint {
			t.Errorf("Decode 1 (uint): Expected %x but got %x", expectedBigInt.Uint64(), actualUint)
			return
		}
	}

	if len(expectedWords) > 1 {
		return
	}

	expectedUint := expectedWords[0]

	actualUint, actualBigInt, actualByteCount, err = Decode(buff)
	if err != nil {
		t.Error(err)
		return
	}
	if actualByteCount != expectedByteCount {
		t.Errorf("Decode 2: Expected byte count of %v but got %v", expectedByteCount, actualByteCount)
		return
	}
	if expectedUint != actualUint {
		t.Errorf("Decode 2: Expected %x but got %x", expectedBigInt.Uint64(), actualUint)
		return
	}
}

func assertEncodeFails(t *testing.T, words []uint64, byteCount int) {
	expectedBigInt := big.NewInt(0)
	expectedBigInt.SetBits(toBigWords(words))
	actualBuffer := &bytes.Buffer{}
	_, err := Encode(expectedBigInt, actualBuffer)
	if err == nil {
		t.Errorf("Expected encoding %v into %v bytes to fail. Result = %v", words, byteCount, actualBuffer.Bytes())
		return
	}
}

func assertDecodeFails(t *testing.T, b ...byte) {
	buff := bytes.NewBuffer(b)
	actualUint, actualBigInt, actualByteCount, err := Decode(buff)
	if err != nil {
		t.Errorf("Expected decoding %v to fail. Result = %v, %v, %v", b, actualUint, actualBigInt, actualByteCount)
		return
	}
}

func TestEncodeDecodeUint(t *testing.T) {
	assertEncodeDecodeUint(t, 0, 0)
	assertEncodeDecodeUint(t, 1, 1)
	assertEncodeDecodeUint(t, 2, 2)
	assertEncodeDecodeUint(t, 0x7f, 0x7f)

	assertEncodeDecodeUint(t, 0x80, 0x80, 0x01)
	assertEncodeDecodeUint(t, 0x81, 0x81, 0x01)
	assertEncodeDecodeUint(t, 0x82, 0x82, 0x01)

	assertEncodeDecodeUint(t, 0x0123456789abcdef, 0xef, 0x9b, 0xaf, 0xcd, 0xf8, 0xac, 0xd1, 0x91, 0x01)
	assertEncodeDecodeUint(t, 0xfedcba9876543210, 0x90, 0xe4, 0xd0, 0xb2, 0x87, 0xd3, 0xae, 0xee, 0xfe, 0x01)
	assertEncodeDecodeUint(t, 0x8000000000000000, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01)
	assertEncodeDecodeUint(t, 0xffffffffffffffff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01)
}

func TestEncodeDecode(t *testing.T) {
	assertEncodeDecode(t, []uint64{0}, 0)
	assertEncodeDecode(t, []uint64{1}, 1)
	assertEncodeDecode(t, []uint64{2}, 2)
	assertEncodeDecode(t, []uint64{0x7f}, 0x7f)

	assertEncodeDecode(t, []uint64{0x80}, 0x80, 0x01)
	assertEncodeDecode(t, []uint64{0x81}, 0x81, 0x01)
	assertEncodeDecode(t, []uint64{0x82}, 0x82, 0x01)

	assertEncodeDecode(t, []uint64{0x0123456789abcdef}, 0xef, 0x9b, 0xaf, 0xcd, 0xf8, 0xac, 0xd1, 0x91, 0x01)
	assertEncodeDecode(t, []uint64{0xfedcba9876543210}, 0x90, 0xe4, 0xd0, 0xb2, 0x87, 0xd3, 0xae, 0xee, 0xfe, 0x01)
	assertEncodeDecode(t, []uint64{0x8000000000000000}, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01)
	assertEncodeDecode(t, []uint64{0xffffffffffffffff}, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01)

	assertEncodeDecode(t, []uint64{0, 1}, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02)
	assertEncodeDecode(t, []uint64{0, 0x8000000000000000},
		0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02)
	assertEncodeDecode(t, []uint64{0xffffffffffffffff, 0xffffffffffffffff},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertEncodeDecode(t, []uint64{0x0123456789abcdef, 0x0123456789abcdef, 0x0123456789abcdef},
		0xef, 0x9b, 0xaf, 0xcd, 0xf8, 0xac, 0xd1, 0x91, 0x81, 0xde, 0xb7, 0xde,
		0x9a, 0xf1, 0xd9, 0xa2, 0xa3, 0x82, 0xbc, 0xef, 0xbc, 0xb5, 0xe2, 0xb3,
		0xc5, 0xc6, 0x04)

	assertEncodeDecode(t, []uint64{0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0x07)
}

func TestExtraData(t *testing.T) {
	var assertExtraData = func(value uint64, expectedByteCount int, b ...byte) {
		buff := bytes.NewBuffer(b)
		actualUint, actualBigInt, actualByteCount, err := Decode(buff)
		if err != nil {
			t.Error(err)
			return
		}
		if actualBigInt != nil {
			t.Errorf("Expected big int to be nil")
		}
		if actualByteCount != expectedByteCount {
			t.Errorf("Expected byte count of 1 but got %v", actualByteCount)
		}
		if actualUint != value {
			t.Errorf("Expected %v but got %v\n", value, actualUint)
		}
	}

	assertExtraData(0, 1, 0x00, 0xff)
	assertExtraData(1, 1, 0x01, 0xff)
	assertExtraData(0x7f, 1, 0x7f, 0x01)
	assertExtraData(0x80, 2, 0x80, 0x01, 0x01, 0x00)
}

// func TestBadData(t *testing.T) {
// 	for i := 0; i < 0x80; i++ {
// 		assertEncodeFails(t, []uint64{uint64(i)}, 0)
// 	}

// 	for i := 0; i < 57; i++ {
// 		word := uint64(0x80) << uint(i)
// 		assertEncodeFails(t, []uint64{word}, 1)
// 	}

// 	for i := 0x80; i < 0x100; i++ {
// 		assertDecodeFails(t, byte(i))
// 	}

// 	for i := 0x80; i < 0x100; i++ {
// 		for j := 0x80; j < 0x100; j++ {
// 			assertDecodeFails(t, byte(i), byte(j))
// 		}
// 	}
// }

func demonstrateUint() {
	v := uint64(104543565)
	buff := &bytes.Buffer{}
	_, err := EncodeUint64(v, buff)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	encoded := buff.Bytes()
	fmt.Printf("%v encodes to %v\n", v, encoded)
	vUint, _, _, err := Decode(buff)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("%v decodes to %v\n", encoded, vUint)
	}
}

func demonstrateBigInt() {
	v := big.NewInt(100000000000000)
	v.Mul(v, v)
	buff := &bytes.Buffer{}
	_, err := Encode(v, buff)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	encoded := buff.Bytes()
	fmt.Printf("%v encodes to %v\n", v, encoded)
	_, vBig, _, err := Decode(buff)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("%v decodes to %v\n", encoded, vBig)
	}
}

func TestDemonstrate(t *testing.T) {
	demonstrateUint()
	demonstrateBigInt()
}
