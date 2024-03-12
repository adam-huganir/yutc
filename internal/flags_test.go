package internal

import (
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var (
	PWD, _                      = os.Getwd()
	REMOVE_WINDOWS_DRIVE_LETTER = regexp.MustCompile(`^[a-zA-Z]:`)
)

func TestParseFileStringFlag(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    *url.URL
		wantErr bool
	}{
		{
			name: "test url",
			args: args{v: "http://example.com"},
			want: &url.URL{
				Scheme:     "http",
				Opaque:     "",
				User:       nil,
				Host:       "example.com",
				Path:       "",
				RawPath:    "",
				ForceQuery: false,
				RawQuery:   "",
				Fragment:   "",
			},
			wantErr: false,
		},
		{
			name: "test local file",
			args: args{v: "./tests/inputData/data1.yaml"},
			want: &url.URL{
				Scheme: "file",
				Opaque: "",
				User:   nil,
				Host:   "",
				// ew
				Path: REMOVE_WINDOWS_DRIVE_LETTER.ReplaceAllString(
					path.Join(
						append(strings.Split(PWD, `\`), "tests/inputData/data1.yaml")...,
					), "",
				),
				RawPath:    "",
				ForceQuery: false,
				RawQuery:   "",
				Fragment:   "",
			},
			wantErr: false,
		},
		{
			name:    "test url with unsupported scheme",
			args:    args{v: "ftp://example.com"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFileStringFlag(tt.args.v)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ParseFileStringFlag() error = %v, wantErr %v", err, tt.wantErr)
				} else if err.Error() != "unsupported scheme, ftp, for url: ftp://example.com" {
					t.Errorf("ParseFileStringFlag() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseFileStringFlag() got = %v, want %v", got, tt.want)
			}
		})
	}
}
