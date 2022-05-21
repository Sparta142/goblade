package ffxiv_test

import (
	"testing"
	"time"

	"github.com/sparta142/goblade/ffxiv"
	"github.com/stretchr/testify/assert"
)

var uncompressedBundleData = []byte{
	0x52, 0x52, 0xa0, 0x41, 0xff, 0x5d, 0x46, 0xe2,
	0x7f, 0x2a, 0x64, 0x4d, 0x7b, 0x99, 0xc4, 0x75,
	0x53, 0xfe, 0xa8, 0x30, 0x7a, 0x01, 0x00, 0x00,
	0x20, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xf8, 0x00, 0x00, 0x00, 0x63, 0x25, 0x6d, 0x10,
	0x63, 0x25, 0x6d, 0x10, 0x03, 0x00, 0x00, 0x00,
	0x14, 0x00, 0x9c, 0x00, 0x00, 0x00, 0x22, 0x02,
	0xa3, 0x10, 0xd1, 0x60, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x20, 0x00, 0x80, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x02, 0x22, 0x53, 0x6f, 0x6d, 0x65, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x20, 0x6d, 0x79, 0x20,
	0x67, 0x65, 0x6e, 0x69, 0x75, 0x73, 0x20, 0x69,
	0x73, 0x2e, 0x2e, 0x2e, 0x20, 0x69, 0x74, 0x27,
	0x73, 0x20, 0x61, 0x6c, 0x6d, 0x6f, 0x73, 0x74,
	0x20, 0x66, 0x72, 0x69, 0x67, 0x68, 0x74, 0x65,
	0x6e, 0x69, 0x6e, 0x67, 0x2e, 0x22, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

var compressedBundleData = []byte{
	0x52, 0x52, 0xa0, 0x41, 0xff, 0x5d, 0x46, 0xe2,
	0x7f, 0x2a, 0x64, 0x4d, 0x7b, 0x99, 0xc4, 0x75,
	0xe8, 0x00, 0xa9, 0x30, 0x7a, 0x01, 0x00, 0x00,
	0x0a, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x78, 0x9c, 0x7b, 0xc1, 0xcc, 0xc0, 0x90, 0xac,
	0x9a, 0x2b, 0x00, 0xc2, 0x40, 0x26, 0x83, 0x08,
	0x43, 0x3f, 0x90, 0x52, 0x62, 0x5a, 0x22, 0x70,
	0x31, 0x81, 0x01, 0x0e, 0x2c, 0x19, 0xf0, 0x01,
	0x26, 0x0c, 0x06, 0x0e, 0x3e, 0x14, 0x38, 0xa0,
	0xd1, 0x02, 0x68, 0xca, 0x59, 0xf0, 0xda, 0x86,
	0x50, 0x87, 0x6e, 0xbc, 0x03, 0x1a, 0x5f, 0x01,
	0x8d, 0x2f, 0x80, 0x46, 0xc3, 0xf4, 0xa3, 0xbb,
	0x07, 0x5d, 0x3d, 0x4c, 0x1c, 0x97, 0xbd, 0x30,
	0x3e, 0xba, 0x7a, 0x16, 0x34, 0x1a, 0xdd, 0x5e,
	0x06, 0x1c, 0xf2, 0x30, 0xba, 0x01, 0x87, 0xb9,
	0x0e, 0x68, 0xe2, 0xe8, 0xfe, 0x60, 0x41, 0xe3,
	0x33, 0xe0, 0xe0, 0x13, 0x03, 0x38, 0x18, 0x18,
	0xc1, 0x74, 0x1f, 0x23, 0xaa, 0x78, 0x2e, 0x8e,
	0xb8, 0x95, 0x60, 0x84, 0x48, 0xbc, 0x61, 0x82,
	0xd0, 0x62, 0x50, 0x3e, 0xcc, 0x1c, 0x76, 0x5c,
	0x89, 0x02, 0x0a, 0x98, 0xa0, 0xea, 0x60, 0x34,
	0x0c, 0xbc, 0x63, 0x42, 0xd5, 0xb7, 0x09, 0x8d,
	0xff, 0x18, 0xca, 0xb7, 0x86, 0x9a, 0xcf, 0x08,
	0xd5, 0xff, 0x1d, 0x2a, 0x2e, 0x8d, 0xa6, 0x7e,
	0x3d, 0x94, 0xff, 0x01, 0x4a, 0xe3, 0xb2, 0x17,
	0xc6, 0xdf, 0x8a, 0xa6, 0x5e, 0x06, 0xea, 0x2f,
	0x49, 0xa8, 0x7d, 0xbf, 0xd1, 0xcc, 0x81, 0x01,
	0x4f, 0xa8, 0xb8, 0x14, 0x94, 0x86, 0x99, 0xc3,
	0x0d, 0x55, 0x67, 0x0f, 0xd5, 0xcf, 0xc7, 0xcc,
	0x84, 0xe2, 0xcf, 0xd9, 0x4c, 0xa8, 0xfe, 0x80,
	0x89, 0xb3, 0xc1, 0xc2, 0x93, 0x19, 0xd5, 0x3f,
	0x7c, 0xcc, 0xf8, 0xc3, 0x15, 0x00, 0x64, 0x23,
	0x19, 0x43,
}

func TestUnmarshalBinary_CompressedIpc(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	var bundle ffxiv.Bundle
	err := bundle.UnmarshalBinary(compressedBundleData)
	assert.NoError(err)

	// Test the header fields
	assert.Equal(ffxiv.IpcMagicBytes, bundle.Magic[:]) // Should be enforced by UnmarshalBinary()
	assert.EqualValues(1624314020072, bundle.Epoch)
	assert.EqualValues(266, bundle.Length)
	assert.EqualValues(0, bundle.ConnectionType)
	assert.EqualValues(1, bundle.Encoding)
	assert.EqualValues(1, bundle.Compression)

	assert.True(bundle.IsCompressed())

	nsec := int((72 * time.Millisecond).Nanoseconds()) // 72 ms
	assert.Equal(time.Date(2021, 6, 21, 22, 20, 20, nsec, time.UTC), bundle.Time())

	// Test the segments slice
	assert.Len(bundle.Segments, 1)

	// Test the first (and hopefully only) segment
	segment := bundle.Segments[0]
	assert.EqualValues(1000, segment.Length)
	assert.EqualValues(0x106d2563, segment.Source)
	assert.EqualValues(0x106d2563, segment.Target)
	assert.EqualValues(3, segment.Type)

	// Test that the first segment is an Ipc
	assert.IsType((*ffxiv.Ipc)(nil), segment.Payload)
	ipc := segment.Payload.(*ffxiv.Ipc)

	// Test the IPC fields
	assert.EqualValues(0x0014, ipc.Magic)
	assert.EqualValues(0x038f, ipc.Type)
	assert.EqualValues(0x0222, ipc.ServerID)
	assert.EqualValues(1624314020, ipc.Epoch)
}

func TestUnmarshalBinary_NonCompressedIpc(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	var bundle ffxiv.Bundle
	err := bundle.UnmarshalBinary(uncompressedBundleData)
	assert.NoError(err)

	// Test the header fields
	assert.Equal(ffxiv.IpcMagicBytes, bundle.Magic[:]) // Should be enforced by UnmarshalBinary()
	assert.EqualValues(1624314019411, bundle.Epoch)
	assert.EqualValues(288, bundle.Length)
	assert.EqualValues(0, bundle.ConnectionType)
	assert.EqualValues(1, bundle.Encoding)
	assert.EqualValues(0, bundle.Compression)

	assert.False(bundle.IsCompressed())

	nsec := int((411 * time.Millisecond).Nanoseconds()) // 411 ms
	assert.Equal(time.Date(2021, 6, 21, 22, 20, 19, nsec, time.UTC), bundle.Time())

	// Test the segments slice
	assert.Len(bundle.Segments, 1)

	// Test the first (and hopefully only) segment
	segment := bundle.Segments[0]
	assert.EqualValues(248, segment.Length)
	assert.EqualValues(0x106d2563, segment.Source)
	assert.EqualValues(0x106d2563, segment.Target)
	assert.EqualValues(3, segment.Type)

	// Test that the first segment is an Ipc
	assert.IsType((*ffxiv.Ipc)(nil), segment.Payload)
	ipc := segment.Payload.(*ffxiv.Ipc)

	// Test the IPC fields
	assert.EqualValues(0x0014, ipc.Magic)
	assert.EqualValues(0x009c, ipc.Type)
	assert.EqualValues(0x0222, ipc.ServerID)
	assert.EqualValues(1624314019, ipc.Epoch)
}

func Benchmark_Bundle_UnmarshalBinary_Uncompressed(b *testing.B) {
	var bundle ffxiv.Bundle
	for n := 0; n < b.N; n++ {
		_ = bundle.UnmarshalBinary(uncompressedBundleData)
	}
}

func Benchmark_Bundle_UnmarshalBinary_Compressed(b *testing.B) {
	var bundle ffxiv.Bundle
	for n := 0; n < b.N; n++ {
		_ = bundle.UnmarshalBinary(compressedBundleData)
	}
}
