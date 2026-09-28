package handlers

import "testing"

func TestDetectContentTypeMobileVideo(t *testing.T) {
	box := func(brand string) []byte {
		b := []byte{0, 0, 0, 0x14, 'f', 't', 'y', 'p'}
		b = append(b, brand...)
		return append(b, make([]byte, 16)...)
	}
	cases := map[string]string{
		"qt  ": "video/quicktime",
		"isom": "video/mp4",
		"heic": "application/octet-stream",
	}
	for brand, want := range cases {
		if got := detectContentType(box(brand)); got != want {
			t.Errorf("brand %q: got %q, want %q", brand, got, want)
		}
	}
}
