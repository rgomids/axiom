package portableconfig_test

import (
	"github.com/rgomids/axiom/internal/portableconfig"
	"testing"
)

func TestPortableProseStructuralSafety(t *testing.T) {
	for _, value := range []string{
		"[Docs](https://example.com);password=synthetic", "[Docs](https://example.com),/home/user/private", "path:/home/user/private", "location:~/private", `Read path:C:\Users\user\private`, "Use `token=synthetic`", "See `/home/user/private`", "See `file:/private/document`", "Use **password=synthetic**",
		"See https://example.com/docs?q=a,b&token=synthetic", "See https://example.com/docs?q=a;b&token=synthetic",
		`{"api_key": "synthetic"}`, "Use password: synthetic", "path=/home/user/private",
		"Read (file:///tmp/document).", "Read %66ile%3A/private/document",
		"See https://example.com?%61pi_key=synthetic", "https://user:synthetic@example.com/doc",
		"See /Users/user/private", `See \\server\share`, "Read ~/private",
		"text\x00more", "text\rmore", "text%00more", "%252525252525252525252525252Fhome",
	} {
		t.Run(value, func(t *testing.T) {
			if portableconfig.SafeProse(value) {
				t.Fatal("unsafe prose accepted")
			}
		})
	}
	for _, value := range []string{
		"A bounded unit of work tracked by the Project.", "Orders.\nRefunds belong to payments.",
		"See https://example.com/docs?lang=en&view=full", "See https://example.com/docs?q=a,b&lang=en", "Read [docs](https://example.com/docs?lang=en).", "See https://example.com/docs?next=/orders", "https://example.com/a,/b", "https://example.com/a;/b", "Read [docs](https://example.com/a_(b))", "[A](https://example.com),[B](https://example.org)", "See https://example.com/a%20b",
		"20% complete, 100% useful", "The password policy protects users.", "The token identifies a request.",
	} {
		t.Run(value, func(t *testing.T) {
			if !portableconfig.SafeProse(value) {
				t.Fatal("normal prose rejected")
			}
		})
	}
}
