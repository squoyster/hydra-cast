package app

import "testing"

func TestExtractReelURLs(t *testing.T) {
	body := `<a href="https://www.facebook.com/page/reel/1111111111/?type=1">x</a>` +
		`<a href='https://www.facebook.com/other/reel/2222222222'>y</a>` +
		`"shareURL":"https://www.facebook.com/page/reel/1111111111/"` // dup

	got := extractReelURLs(body, "")
	want := map[string]bool{
		"https://www.facebook.com/page/reel/1111111111": true,
		"https://www.facebook.com/other/reel/2222222222": true,
	}

	if len(got) != len(want) {
		t.Fatalf("got %d URLs %v, want %d", len(got), got, len(want))
	}
	for _, u := range got {
		if !want[u] {
			t.Errorf("unexpected URL %q", u)
		}
	}
}

func TestReelExternalID(t *testing.T) {
	cases := map[string]string{
		"https://www.facebook.com/x/reel/1234567890":  "1234567890",
		"https://www.facebook.com/x/reel/1234567890/": "1234567890",
		"9876543210":                                  "9876543210",
	}
	for in, want := range cases {
		if got := reelExternalID(in); got != want {
			t.Errorf("reelExternalID(%q) = %q, want %q", in, got, want)
		}
	}
}
