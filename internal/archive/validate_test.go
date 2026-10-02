package archive

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestAdmissionRejectsMetadataLargerThanDiskBudget(t *testing.T) {
	var data bytes.Buffer

	writer := zip.NewWriter(&data)
	for _, name := range []string{"first", "second", "third"} {
		entry, err := writer.CreateRaw(
			&zip.FileHeader{Name: name, Method: zip.Store, CompressedSize64: 1, UncompressedSize64: 200 * 1024 * 1024},
		)
		if err != nil {
			t.Fatal(err)
		}

		if _, writeErr := entry.Write([]byte{'x'}); writeErr != nil {
			t.Fatal(writeErr)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	if err := Validate(bytes.NewReader(data.Bytes()), int64(data.Len())); err == nil {
		t.Fatal("600 MiB archive exceeds reserved512 MiB site budget")
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
