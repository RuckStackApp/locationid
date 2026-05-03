package locationid

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

type testVector struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Precision uint    `json:"precision"`
	Code      string  `json:"code"`
}

func TestNew(t *testing.T) {
	code := New("8f3k9zdq2ma")

	if got := code.String(); got != "8F3K9ZDQ2MA" {
		t.Fatalf("String() = %q, want %q", got, "8F3K9ZDQ2MA")
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	code, err := Encode(37.7749, -122.4194, 10)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	decoded, err := Decode(code)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if decoded.Code != code {
		t.Fatalf("decoded.Code = %q, want %q", decoded.Code, code)
	}

	if decoded.Precision != 10 {
		t.Fatalf("decoded.Precision = %d, want 10", decoded.Precision)
	}

	if len(decoded.Payload) != 4 {
		t.Fatalf("len(decoded.Payload) = %d, want 4", len(decoded.Payload))
	}

	if decoded.Bounds.MinLat > 37.7749 || decoded.Bounds.MaxLat < 37.7749 {
		t.Fatal("decoded latitude bounds do not contain the source point")
	}

	if decoded.Bounds.MinLon > -122.4194 || decoded.Bounds.MaxLon < -122.4194 {
		t.Fatal("decoded longitude bounds do not contain the source point")
	}
}

func TestEncodeAtWorldCenter(t *testing.T) {
	code, err := Encode(0, 0, 10)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if got := code.String(); got != "R000A" {
		t.Fatalf("Encode() = %q, want %q", got, "R000A")
	}
}

func TestParent(t *testing.T) {
	code, err := Encode(37.7749, -122.4194, 10)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	parent, err := Parent(code, 8)
	if err != nil {
		t.Fatalf("Parent() error = %v", err)
	}

	decodedParent, err := Decode(parent)
	if err != nil {
		t.Fatalf("Decode(parent) error = %v", err)
	}

	decodedChild, err := Decode(code)
	if err != nil {
		t.Fatalf("Decode(child) error = %v", err)
	}

	if decodedParent.Precision != 8 {
		t.Fatalf("parent precision = %d, want 8", decodedParent.Precision)
	}

	if decodedParent.Bounds.MinLat > decodedChild.CenterLat || decodedParent.Bounds.MaxLat < decodedChild.CenterLat {
		t.Fatal("parent latitude bounds do not contain child center")
	}

	if decodedParent.Bounds.MinLon > decodedChild.CenterLon || decodedParent.Bounds.MaxLon < decodedChild.CenterLon {
		t.Fatal("parent longitude bounds do not contain child center")
	}
}

func TestPrecisionAndPayload(t *testing.T) {
	code := LocationCode("8F3K9ZDQ2MA")

	precision, err := code.Precision()
	if err != nil {
		t.Fatalf("Precision() error = %v", err)
	}

	if precision != 10 {
		t.Fatalf("Precision() = %d, want 10", precision)
	}

	if payload := code.Payload(); payload != "8F3K9ZDQ2M" {
		t.Fatalf("Payload() = %q, want %q", payload, "8F3K9ZDQ2M")
	}
}

func TestEncodeRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		err  error
		lat  float64
		lon  float64
		prec uint
	}{
		{name: "latitude", err: ErrInvalidLatitude, lat: 91, lon: 0, prec: 10},
		{name: "longitude", err: ErrInvalidLongitude, lat: 0, lon: 181, prec: 10},
		{name: "precision", err: ErrInvalidPrecision, lat: 0, lon: 0, prec: 32},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Encode(test.lat, test.lon, test.prec)
			if !errors.Is(err, test.err) {
				t.Fatalf("Encode() error = %v, want %v", err, test.err)
			}
		})
	}
}

func TestDecodeRejectsInvalidCode(t *testing.T) {
	tests := []LocationCode{
		"",
		"abc",
		"R000I",
		"TOOLONGA",
	}

	for _, code := range tests {
		t.Run(strings.ReplaceAll(code.String(), "", "_"), func(t *testing.T) {
			_, err := Decode(code)
			if err == nil {
				t.Fatal("Decode() error = nil, want non-nil")
			}
		})
	}
}

func TestSharedSpecVectors(t *testing.T) {
	data, err := os.ReadFile("../spec/test-vectors.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var vectors []testVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	for _, vector := range vectors {
		code, err := Encode(vector.Lat, vector.Lon, vector.Precision)
		if err != nil {
			t.Fatalf("Encode(%v, %v, %d) error = %v", vector.Lat, vector.Lon, vector.Precision, err)
		}

		if code.String() != vector.Code {
			t.Fatalf("Encode(%v, %v, %d) = %q, want %q", vector.Lat, vector.Lon, vector.Precision, code, vector.Code)
		}
	}
}
