Unsigned Little Endian Base-128
===============================

This is a go implementation of [Unsigned LEB128](https://en.wikipedia.org/wiki/LEB128),
with support for encoding and decoding uint64 and unsigned math.big.Int values.


Usage
-----

```golang
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
```

Prints:

```
104543565 encodes to [205 234 236 49]
[205 234 236 49] decodes to 104543565
```


```golang
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
```

Prints:

```
10000000000000000000000000000 encodes to [128 128 128 128 145 204 192 146 190 188 185 254 132 4]
[128 128 128 128 145 204 192 146 190 188 185 254 132 4] decodes to 10000000000000000000000000000
```


License
-------

MIT License:

Copyright 2020 Karl Stenerud

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
