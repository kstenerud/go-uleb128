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
	actualFilledByteCount := Encode(expectedBigInt, actualBytes)
	if actualFilledByteCount != expectedByteCount {
		t.Errorf("Expected byte count of %v but got %v", expectedByteCount, actualFilledByteCount)
	}
	if !reflect.DeepEqual(actualBytes, expectedBytes) {
		t.Errorf("Expected %v but got %v", describe.D(expectedBytes), describe.D(actualBytes))
	}
	actualUint, actualBigInt, actualByteCount, err := Decode(actualBytes)
	if err != nil {
		t.Error(err)
	}
	if actualByteCount != expectedByteCount {
		t.Errorf("Expected byte count of %v but got %v", expectedByteCount, actualByteCount)
	}
	if len(expectedBigInt.Bits()) == 1 {
		if expectedBigInt.Uint64() != actualUint {
			t.Errorf("Expected %x but got %x", expectedBigInt.Uint64(), actualUint)
		}
	} else {
		if expectedBigInt.Cmp(actualBigInt) != 0 {
			t.Errorf("Expected %x but got %x", expectedBigInt, actualBigInt)
		}
	}

	if len(words) > 1 {
		return
	}

	expectedUint := words[0]
	expectedByteCount = EncodedSizeUint64(expectedUint)
	actualBytes = make([]byte, expectedByteCount, expectedByteCount)

	actualFilledByteCount = EncodeUint64(expectedUint, actualBytes)
	if actualFilledByteCount != expectedByteCount {
		t.Errorf("Expected byte count of %v but got %v", expectedByteCount, actualFilledByteCount)
	}

	if !reflect.DeepEqual(actualBytes, expectedBytes) {
		t.Errorf("Expected %v but got %v", describe.D(expectedBytes), describe.D(actualBytes))
	}
	actualUint, actualBigInt, actualByteCount, err = Decode(actualBytes)
	if err != nil {
		t.Error(err)
	}
	if actualByteCount != expectedByteCount {
		t.Errorf("Expected byte count of %v but got %v", expectedByteCount, actualByteCount)
	}
	if expectedUint != actualUint {
		t.Errorf("Expected %x but got %x", expectedBigInt.Uint64(), actualUint)
	}
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

func demonstrateUint() {
	v := uint64(104543565)
	encodedSize := EncodedSizeUint64(v)
	bytes := make([]byte, encodedSize, encodedSize)
	EncodeUint64(v, bytes)
	fmt.Printf("%v encodes to %v\n", v, bytes)
	vUint, _, _, err := Decode(bytes)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("%v decodes to %v\n", bytes, vUint)
	}
}

func demonstrateBigInt() {
	v := big.NewInt(100000000000000)
	v.Mul(v, v)
	encodedSize := EncodedSize(v)
	bytes := make([]byte, encodedSize, encodedSize)
	Encode(v, bytes)
	fmt.Printf("%v encodes to %v\n", v, bytes)
	_, vBig, _, err := Decode(bytes)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("%v decodes to %v\n", bytes, vBig)
	}
}

func TestDemonstrate(t *testing.T) {
	demonstrateUint()
	demonstrateBigInt()
}
