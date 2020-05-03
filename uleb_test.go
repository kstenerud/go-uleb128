package uleb128

import (
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
	actualBytes := make([]byte, expectedByteCount, expectedByteCount)
	actualFilledByteCount, ok := Encode(expectedBigInt, actualBytes)
	if !ok {
		t.Errorf("Not enough room to encode %v into %v bytes", expectedBigInt, len(actualBytes))
		return
	}
	if actualFilledByteCount != expectedByteCount {
		t.Errorf("Encode 1: Expected byte count of %v but got %v", expectedByteCount, actualFilledByteCount)
		return
	}
	if !reflect.DeepEqual(actualBytes, expectedBytes) {
		t.Errorf("Encode 1: Expected %v but got %v", describe.D(expectedBytes), describe.D(actualBytes))
		return
	}
	actualUint, actualBigInt, actualByteCount, ok := Decode(0, 0, actualBytes)
	if !ok {
		t.Errorf("Decode 1 %v: Unterminated uleb decoding %v", describe.D(actualBytes), actualBytes)
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

	if len(words) > 1 {
		return
	}

	expectedUint := words[0]
	expectedByteCount = EncodedSizeUint64(expectedUint)
	actualBytes = make([]byte, expectedByteCount, expectedByteCount)

	actualFilledByteCount, ok = EncodeUint64(expectedUint, actualBytes)
	if !ok {
		t.Errorf("Encode 2: Not enough room to encode %v into %v bytes", expectedUint, len(actualBytes))
		return
	}
	if actualFilledByteCount != expectedByteCount {
		t.Errorf("Encode 2: Expected byte count of %v but got %v", expectedByteCount, actualFilledByteCount)
		return
	}

	if !reflect.DeepEqual(actualBytes, expectedBytes) {
		t.Errorf("Encode 2: Expected %v but got %v", describe.D(expectedBytes), describe.D(actualBytes))
		return
	}
	actualUint, actualBigInt, actualByteCount, ok = Decode(0, 0, actualBytes)
	if !ok {
		t.Errorf("Decode 2: Unterminated uleb decoding %v", actualBytes)
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
	actualBytes := make([]byte, expectedByteCount, expectedByteCount)
	actualFilledByteCount, ok := EncodeUint64(value, actualBytes)
	if !ok {
		t.Errorf("Not enough room to encode %v into %v bytes", value, len(actualBytes))
		return
	}
	if actualFilledByteCount != expectedByteCount {
		t.Errorf("Encode 1: Expected byte count of %v but got %v", expectedByteCount, actualFilledByteCount)
		return
	}
	if !reflect.DeepEqual(actualBytes, expectedBytes) {
		t.Errorf("Encode 1: Expected %v but got %v", describe.D(expectedBytes), describe.D(actualBytes))
		return
	}
	actualUint, actualBigInt, actualByteCount, ok := Decode(0, 0, actualBytes)
	if !ok {
		t.Errorf("Decode 1 %v: Unterminated uleb decoding %v", describe.D(actualBytes), actualBytes)
		return
	}
	if actualByteCount != expectedByteCount {
		t.Errorf("Decode 1: Expected byte count of %v but got %v", expectedByteCount, actualByteCount)
		return
	}
	if actualBigInt != nil {
		t.Errorf("%v (from %v) should not decode to a big int", describe.D(actualBytes), value)
		return
	}
	if value != actualUint {
		t.Errorf("Decode 1 (uint): Expected %x but got %x", value, actualUint)
		return
	}
}

func assertDecode(t *testing.T, preValue uint64, preBitCount int, expectedWords []uint64, bytes ...byte) {
	expectedBigInt := big.NewInt(0)
	expectedBigInt.SetBits(toBigWords(expectedWords))
	expectedByteCount := len(bytes)

	actualUint, actualBigInt, actualByteCount, ok := Decode(preValue, preBitCount, bytes)
	if !ok {
		t.Errorf("Decode 1: Unterminated uleb decoding %v", bytes)
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

	actualUint, actualBigInt, actualByteCount, ok = Decode(preValue, preBitCount, bytes)
	if !ok {
		t.Errorf("Decode 2: Unterminated uleb decoding %v", bytes)
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
	actualBytes := make([]byte, byteCount, byteCount)
	_, ok := Encode(expectedBigInt, actualBytes)
	if ok {
		t.Errorf("Expected encoding %v into %v bytes to fail. Result = %v", words, byteCount, actualBytes)
		return
	}
}

func assertDecodeFails(t *testing.T, bytes ...byte) {
	actualUint, actualBigInt, actualByteCount, ok := Decode(0, 0, bytes)
	if ok {
		t.Errorf("Expected decoding %v to fail. Result = %v, %v, %v", bytes, actualUint, actualBigInt, actualByteCount)
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

func TestEmptyData(t *testing.T) {
	expectedUint := uint64(0xffffffffffffffff)
	actualUint, actualBigInt, actualByteCount, ok := Decode(expectedUint, 64, []byte{})
	if ok {
		t.Errorf("Should not be ok")
		return
	}
	if actualBigInt != nil {
		t.Errorf("Expected big int to be nil")
	}
	if actualByteCount != 0 {
		t.Errorf("Expected byte count of 0 but got %v", actualByteCount)
	}
	if actualUint != expectedUint {
		t.Errorf("Expected %v but got %v\n", expectedUint, actualUint)
	}
}

func TestExtraData(t *testing.T) {
	var assertExtraData = func(value uint64, expectedByteCount int, bytes ...byte) {
		actualUint, actualBigInt, actualByteCount, ok := Decode(0, 0, bytes)
		if !ok {
			t.Errorf("Decoding failed")
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

func TestPreValues(t *testing.T) {
	assertDecode(t, 1, 1, []uint64{0x03}, 0x01)
	assertDecode(t, 0xff, 1, []uint64{0x03}, 0x01)
	assertDecode(t, 0, 6, []uint64{0xc0}, 0x03)

	for i := 0; i < 63; i++ {
		word := uint64(1) << uint(i)
		assertDecode(t, 0, i, []uint64{word}, 0x01)
	}

	for i := 0; i < 57; i++ {
		word := uint64(0x80) << uint(i)
		assertDecode(t, 0, i, []uint64{word}, 0x80, 0x01)
	}

	assertDecode(t, 0, 0, []uint64{0xffffffffffffffff, 1},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 0, 1, []uint64{0xfffffffffffffffe, 3},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 1, 1, []uint64{0xffffffffffffffff, 3},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 0, 2, []uint64{0xfffffffffffffffc, 7},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 1, 2, []uint64{0xfffffffffffffffd, 7},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 2, 2, []uint64{0xfffffffffffffffe, 7},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 3, 2, []uint64{0xffffffffffffffff, 7},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 0, 3, []uint64{0xfffffffffffffff8, 0x0f},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 0, 4, []uint64{0xfffffffffffffff0, 0x1f},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 0, 5, []uint64{0xffffffffffffffe0, 0x3f},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 0, 6, []uint64{0xffffffffffffffc0, 0x7f},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 0, 7, []uint64{0xffffffffffffff80, 0xff},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)

	assertDecode(t, 0, 8, []uint64{0xffffffffffffff00, 0x1ff},
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x03)
}

func TestBadData(t *testing.T) {
	for i := 0; i < 0x80; i++ {
		assertEncodeFails(t, []uint64{uint64(i)}, 0)
	}

	for i := 0; i < 57; i++ {
		word := uint64(0x80) << uint(i)
		assertEncodeFails(t, []uint64{word}, 1)
	}

	for i := 0x80; i < 0x100; i++ {
		assertDecodeFails(t, byte(i))
	}

	for i := 0x80; i < 0x100; i++ {
		for j := 0x80; j < 0x100; j++ {
			assertDecodeFails(t, byte(i), byte(j))
		}
	}
}

func demonstrateUint() {
	v := uint64(104543565)
	encodedSize := EncodedSizeUint64(v)
	bytes := make([]byte, encodedSize, encodedSize)
	_, ok := EncodeUint64(v, bytes)
	if !ok {
		fmt.Printf("Error: Not enough room to encode %v\n", v)
	}
	fmt.Printf("%v encodes to %v\n", v, bytes)
	vUint, _, _, ok := Decode(0, 0, bytes)
	if !ok {
		fmt.Printf("Error: Unterminated ULEB128: %v\n", bytes)
	} else {
		fmt.Printf("%v decodes to %v\n", bytes, vUint)
	}
}

func demonstrateBigInt() {
	v := big.NewInt(100000000000000)
	v.Mul(v, v)
	encodedSize := EncodedSize(v)
	bytes := make([]byte, encodedSize, encodedSize)
	_, ok := Encode(v, bytes)
	if !ok {
		fmt.Printf("Error: Not enough room to encode %v\n", v)
	}
	fmt.Printf("%v encodes to %v\n", v, bytes)
	_, vBig, _, ok := Decode(0, 0, bytes)
	if !ok {
		fmt.Printf("Error: Unterminated ULEB128: %v\n", bytes)
	} else {
		fmt.Printf("%v decodes to %v\n", bytes, vBig)
	}
}

func demonstratePreValue() {
	bytes := []byte{0xd1, 0xf9, 0xd6, 3}
	preValue := uint64(0x05)
	preBitCount := 4
	vUint, _, _, ok := Decode(preValue, preBitCount, bytes)
	if !ok {
		fmt.Printf("Error: Unterminated ULEB128: %v\n", bytes)
	} else {
		fmt.Printf("%v with %v-bit pre-value (%v) decodes to %v\n", bytes, preBitCount, preValue, vUint)
	}
}

func TestDemonstrate(t *testing.T) {
	demonstrateUint()
	demonstrateBigInt()
	demonstratePreValue()
}
