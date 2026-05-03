# LocationCode Spec

## Goal

`LocationCode` is a deterministic, compact, alphanumeric encoding of a geographic cell.

It is designed for stable location references and parent/child cell math. A `LocationCode` identifies a cell, not an infinitely precise point.

## Public Format

```text
<payload><precision>
```

Example:

```text
8F3K9ZDQ2MA
```

Where:

```text
payload   = 8F3K9ZDQ2M
precision = A
```

## Alphabet

Use Crockford-style Base32:

```text
0123456789ABCDEFGHJKMNPQRSTVWXYZ
```

Excluded characters:

```text
I L O U
```

## Precision

The final character encodes precision using its index in the alphabet.

Examples:

```text
0 => 0
9 => 9
A => 10
Z => 31
```

The precision determines how many spatial bits are meaningful per axis.

## Coordinate Normalization

Latitude and longitude are normalized into integer ranges.

```text
latitude:  -90  to +90
longitude: -180 to +180
```

Normalize:

```go
latNorm = (lat + 90.0) / 180.0
lonNorm = (lon + 180.0) / 360.0
```

Clamp normalized values to `[0.0, 1.0)`.

Reject latitude outside `[-90, 90]` and longitude outside `[-180, 180]`.

## Spatial Integer Encoding

For precision `p`:

```go
bitsPerAxis = p
maxValue = 1 << bitsPerAxis
```

Then:

```go
latInt = floor(latNorm * maxValue)
lonInt = floor(lonNorm * maxValue)
```

Clamp edge cases:

```go
if latInt == maxValue {
    latInt = maxValue - 1
}

if lonInt == maxValue {
    lonInt = maxValue - 1
}
```

## Bit Interleaving

Interleave longitude and latitude bits into a single unsigned integer using Morton/Z-order encoding.

Bit order:

```text
lon bit, lat bit, lon bit, lat bit...
```

From most significant bit to least significant bit.

## Payload Encoding

Encode the interleaved integer as Base32 using the alphabet.

```go
payloadBits = bitsPerAxis * 2
payloadChars = ceil(payloadBits / 5)
```

Left-pad the payload with `0` until it reaches `payloadChars`.

## Full Encoding

Encoding returns:

```text
<payload><precision>
```

Where the final precision character is `Alphabet[precision]`.

## Decoding

Split the code into payload and precision suffix.

Decode the payload from Base32, deinterleave back to `latInt` and `lonInt`, and convert the cell back into geographic bounds.

Return:

```go
type Bounds struct {
    MinLat float64
    MaxLat float64
    MinLon float64
    MaxLon float64
}

type DecodedLocation struct {
    Code      string
    Payload   string
    Precision uint
    Bounds    Bounds
    CenterLat float64
    CenterLon float64
}
```

Center:

```go
centerLat = (latMin + latMax) / 2
centerLon = (lonMin + lonMax) / 2
```

## Parent Cells

A parent cell is a lower-precision cell containing the current cell.

```go
func Parent(code string, parentPrecision uint) string {
    decoded := Decode(code)
    return Encode(decoded.CenterLat, decoded.CenterLon, parentPrecision)
}
```
