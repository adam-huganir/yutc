package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsArchive(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{"tar.gz", "test.tar.gz", true},
		{"tgz", "test.tgz", true},
		{"tar", "test.tar", true},
		{"zip", "test.zip", true},
		{"gz", "test.gz", true},
		{"txt", "test.txt", false},
		{"yaml", "test.yaml", false},
		{"no extension", "test", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsArchive(tt.filePath))
		})
	}
}
