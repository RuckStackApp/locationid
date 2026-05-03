package locationid

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

const Alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

const maxPrecision = 31

var decodeMap = func() map[rune]uint {
	index := make(map[rune]uint, len(Alphabet))
	for i, char := range Alphabet {
		index[char] = uint(i)
	}
	return index
}()

var (
	ErrInvalidLatitude  = errors.New("latitude must be within [-90, 90]")
	ErrInvalidLongitude = errors.New("longitude must be within [-180, 180]")
	ErrInvalidPrecision = fmt.Errorf("precision must be within [0, %d]", maxPrecision)
	ErrInvalidCode      = errors.New("invalid location code")
)

// LocationCode identifies a geographic cell.
type LocationCode string

// LocationID is kept as an alias for the package's public identifier type.
type LocationID = LocationCode

type Bounds struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

type DecodedLocation struct {
	Code      LocationCode
	Payload   string
	Precision uint
	Bounds    Bounds
	CenterLat float64
	CenterLon float64
}

// New returns a LocationCode from the provided value.
func New(value string) LocationCode {
	return LocationCode(strings.ToUpper(value))
}

// String returns the string form of the location code.
func (code LocationCode) String() string {
	return string(code)
}

// IsZero reports whether the location code is empty.
func (code LocationCode) IsZero() bool {
	return code == ""
}

// Payload returns the code payload without the precision suffix.
func (code LocationCode) Payload() string {
	if len(code) == 0 {
		return ""
	}

	return string(code[:len(code)-1])
}

// Precision returns the decoded precision from the final character.
func (code LocationCode) Precision() (uint, error) {
	if len(code) == 0 {
		return 0, ErrInvalidCode
	}

	precision, ok := decodeMap[rune(code[len(code)-1])]
	if !ok {
		return 0, fmt.Errorf("%w: unknown precision character %q", ErrInvalidCode, code[len(code)-1])
	}

	return precision, nil
}

// Encode converts latitude and longitude into a location code at the requested precision.
func Encode(lat, lon float64, precision uint) (LocationCode, error) {
	if lat < -90 || lat > 90 {
		return "", ErrInvalidLatitude
	}

	if lon < -180 || lon > 180 {
		return "", ErrInvalidLongitude
	}

	if precision > maxPrecision {
		return "", ErrInvalidPrecision
	}

	latNorm := clampUnit((lat + 90.0) / 180.0)
	lonNorm := clampUnit((lon + 180.0) / 360.0)

	maxValue := uint64(1) << precision
	latInt := normalizedToInt(latNorm, maxValue)
	lonInt := normalizedToInt(lonNorm, maxValue)

	z := interleave(latInt, lonInt, precision)
	payloadChars := ceilDiv(precision*2, 5)
	payload := encodeBase32(z)
	payload = leftPad(payload, int(payloadChars), '0')

	return LocationCode(payload + string(Alphabet[precision])), nil
}

// Decode parses a location code and returns its spatial bounds and center.
func Decode(code LocationCode) (DecodedLocation, error) {
	if len(code) == 0 {
		return DecodedLocation{}, ErrInvalidCode
	}

	precision, err := code.Precision()
	if err != nil {
		return DecodedLocation{}, err
	}

	payload := code.Payload()
	expectedPayloadChars := int(ceilDiv(precision*2, 5))
	if len(payload) != expectedPayloadChars {
		return DecodedLocation{}, fmt.Errorf("%w: payload length %d does not match precision %d", ErrInvalidCode, len(payload), precision)
	}

	z, err := decodeBase32(payload)
	if err != nil {
		return DecodedLocation{}, err
	}

	latInt, lonInt := deinterleave(z, precision)
	maxValue := float64(uint64(1) << precision)

	latMinNorm := float64(latInt) / maxValue
	latMaxNorm := float64(latInt+1) / maxValue
	lonMinNorm := float64(lonInt) / maxValue
	lonMaxNorm := float64(lonInt+1) / maxValue

	bounds := Bounds{
		MinLat: latMinNorm*180.0 - 90.0,
		MaxLat: latMaxNorm*180.0 - 90.0,
		MinLon: lonMinNorm*360.0 - 180.0,
		MaxLon: lonMaxNorm*360.0 - 180.0,
	}

	return DecodedLocation{
		Code:      code,
		Payload:   payload,
		Precision: precision,
		Bounds:    bounds,
		CenterLat: (bounds.MinLat + bounds.MaxLat) / 2,
		CenterLon: (bounds.MinLon + bounds.MaxLon) / 2,
	}, nil
}

// Parent returns the lower-precision cell that contains the code.
func Parent(code LocationCode, parentPrecision uint) (LocationCode, error) {
	decoded, err := Decode(code)
	if err != nil {
		return "", err
	}

	if parentPrecision > decoded.Precision {
		return "", fmt.Errorf("%w: parent precision %d exceeds code precision %d", ErrInvalidPrecision, parentPrecision, decoded.Precision)
	}

	return Encode(decoded.CenterLat, decoded.CenterLon, parentPrecision)
}

func clampUnit(value float64) float64 {
	if value < 0 {
		return 0
	}

	if value >= 1 {
		return math.Nextafter(1, 0)
	}

	return value
}

func normalizedToInt(value float64, maxValue uint64) uint64 {
	if maxValue == 1 {
		return 0
	}

	result := uint64(math.Floor(value * float64(maxValue)))
	if result >= maxValue {
		return maxValue - 1
	}

	return result
}

func interleave(latInt, lonInt uint64, bitsPerAxis uint) uint64 {
	var z uint64

	for i := bitsPerAxis; i > 0; i-- {
		bit := i - 1
		lonBit := (lonInt >> bit) & 1
		latBit := (latInt >> bit) & 1

		z = (z << 1) | lonBit
		z = (z << 1) | latBit
	}

	return z
}

func deinterleave(z uint64, bitsPerAxis uint) (uint64, uint64) {
	var latInt uint64
	var lonInt uint64

	for i := uint(0); i < bitsPerAxis; i++ {
		shift := (bitsPerAxis - 1 - i) * 2
		lonBit := (z >> (shift + 1)) & 1
		latBit := (z >> shift) & 1

		lonInt = (lonInt << 1) | lonBit
		latInt = (latInt << 1) | latBit
	}

	return latInt, lonInt
}

func encodeBase32(value uint64) string {
	if value == 0 {
		return "0"
	}

	encoded := make([]byte, 0, 13)
	for value > 0 {
		encoded = append(encoded, Alphabet[value%32])
		value /= 32
	}

	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}

	return string(encoded)
}

func decodeBase32(value string) (uint64, error) {
	if value == "" {
		return 0, nil
	}

	var decoded uint64
	for _, char := range value {
		index, ok := decodeMap[char]
		if !ok {
			return 0, fmt.Errorf("%w: invalid payload character %q", ErrInvalidCode, char)
		}

		decoded = decoded*32 + uint64(index)
	}

	return decoded, nil
}

func ceilDiv(a, b uint) uint {
	if a == 0 {
		return 0
	}

	return (a + b - 1) / b
}

func leftPad(value string, length int, pad byte) string {
	if len(value) >= length {
		return value
	}

	padded := make([]byte, length)
	padLength := length - len(value)
	for i := 0; i < padLength; i++ {
		padded[i] = pad
	}
	copy(padded[padLength:], value)

	return string(padded)
}
