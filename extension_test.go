package pgxman

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtension_ParseSource(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name    string
		ext     Extension
		wantExt ExtensionSource
		wantErr bool
	}{
		{
			name: "http source",
			ext: Extension{
				Source: "http://example.com/test.tar.gz",
			},
			wantExt: &httpExtensionSource{URL: "http://example.com/test.tar.gz"},
		},
		{
			name: "file source",
			ext: Extension{
				Source: "file:///tmp/test.tar.gz",
			},
			wantExt: &fileExtensionSource{Dir: "/tmp/test.tar.gz"},
		},
		{
			name: "invalid source",
			ext: Extension{
				Source: "ftp://example.com/test.tar.gz",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotExt, gotErr := tt.ext.ParseSource()

			assert.Equal(tt.wantErr, gotErr != nil)
			assert.Equal(tt.wantExt, gotExt)
		})
	}
}
