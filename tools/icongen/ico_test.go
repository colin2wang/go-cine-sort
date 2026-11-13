package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func testPNG(size int) []byte {
	// The ICO encoder only treats frames as opaque bytes; a tiny marker is
	// enough to validate offsets and directory entries.
	return bytes.Repeat([]byte{byte(size)}, 64+size)
}

func TestEncodeICO(t *testing.T) {
	frames := []ICOImage{
		{Size: 16, PNG: testPNG(16)},
		{Size: 256, PNG: testPNG(256)},
	}
	data, err := EncodeICO(frames)
	if err != nil {
		t.Fatal(err)
	}
	le16 := func(offset int) uint16 { return binary.LittleEndian.Uint16(data[offset:]) }
	le32 := func(offset int) uint32 { return binary.LittleEndian.Uint32(data[offset:]) }

	if le16(0) != 0 || le16(2) != 1 || le16(4) != 2 {
		t.Fatalf("ICONDIR header invalid: %v %v %v", le16(0), le16(2), le16(4))
	}
	// First entry starts at byte 6.
	if data[6] != 16 || data[7] != 16 || data[8] != 0 || data[9] != 0 {
		t.Errorf("first ICONDIRENTRY dimension bytes invalid: %v", data[6:10])
	}
	if le16(10) != 1 || le16(12) != 32 {
		t.Errorf("planes/bitcount invalid: %d/%d", le16(10), le16(12))
	}
	if got, want := le32(14), uint32(len(frames[0].PNG)); got != want {
		t.Errorf("first image size = %d, want %d", got, want)
	}
	expectedOffset := uint32(6 + 16*len(frames))
	if got := le32(18); got != expectedOffset {
		t.Errorf("first image offset = %d, want %d", got, expectedOffset)
	}
	// Second entry: 256 is encoded as 0.
	if data[22] != 0 || data[23] != 0 {
		t.Errorf("256px dimension should be encoded as 0, got %d/%d", data[22], data[23])
	}
	if got, want := le32(34), expectedOffset+uint32(len(frames[0].PNG)); got != want {
		t.Errorf("second image offset = %d, want %d", got, want)
	}
	if !bytes.Equal(data[len(data)-len(frames[1].PNG):], frames[1].PNG) {
		t.Error("last PNG payload was not written verbatim")
	}
	if got, want := len(data), 6+16*2+len(frames[0].PNG)+len(frames[1].PNG); got != want {
		t.Errorf("total size = %d, want %d", got, want)
	}
}

func TestEncodeICOValidation(t *testing.T) {
	if _, err := EncodeICO(nil); err == nil {
		t.Error("empty frame list should fail")
	}
	if _, err := EncodeICO([]ICOImage{{Size: 512, PNG: testPNG(1)}}); err == nil {
		t.Error("size > 256 should fail")
	}
	if _, err := EncodeICO([]ICOImage{{Size: 32, PNG: nil}}); err == nil {
		t.Error("empty PNG should fail")
	}
}
