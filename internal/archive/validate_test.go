package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"testing"
)

func TestAdmissionRejectsMetadataLargerThanDiskBudget(t *testing.T) {
	var data bytes.Buffer

	const entrySize = 200 * 1024 * 1024
	// Six MiB of raw ZIP data keeps the advertised 600 MiB expansion within
	// the ratio limit, so rejection must come from the aggregate disk budget.
	compressed := make([]byte, entrySize/MaxExpansionRatio)

	writer := zip.NewWriter(&data)
	for _, name := range []string{"first", "second", "third"} {
		entry, err := writer.CreateRaw(
			&zip.FileHeader{
				Name: name, Method: zip.Deflate,
				CompressedSize64: uint64(len(compressed)), UncompressedSize64: entrySize,
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		if _, writeErr := entry.Write(compressed); writeErr != nil {
			t.Fatal(writeErr)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf("ZIP too large: %d bytes uncompressed exceeds 512 MiB limit", 3*entrySize)
	if err := Validate(bytes.NewReader(data.Bytes()), int64(data.Len())); err == nil || err.Error() != want {
		t.Fatalf("600 MiB metadata must exceed the aggregate disk budget: got %v, want %q", err, want)
	}
}

func TestMalformedAndOversizedCompressedInput(t *testing.T) {
	for name, size := range map[string]int64{"negative": -1, "oversized": MaxCompressedSize + 1, "malformed": 3} {
		t.Run(name, func(t *testing.T) {
			if err := Validate(bytes.NewReader([]byte("zip")), size); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}
