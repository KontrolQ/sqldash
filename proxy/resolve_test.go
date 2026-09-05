package proxy

import (
	"net/http"
	"testing"

	"sqldash/config"
)

func TestTheHostNamesTheDatabaseNotTheClient(t *testing.T) {
	config.Server.Domain = "db.example.com"

	cases := []struct {
		what      string
		host      string
		namespace string
		wanted    string
	}{
		{"a plain subdomain", "software.db.example.com", "", "software"},
		{"a client asking for something else", "software.db.example.com", "default", "software"},
		{"a client asking for another database", "software.db.example.com", "secrets", "software"},
		{"the panel itself", "db.example.com", "", ""},
		{"the panel with a header", "db.example.com", "chosen", "chosen"},
		{"a deeper name", "one.two.db.example.com", "", ""},
		{"a port", "software.db.example.com:8443", "", "software"},
		{"mixed case", "SOFTWARE.DB.EXAMPLE.COM", "", "software"},
		{"a stranger", "elsewhere.example.org", "", ""},
	}

	for _, held := range cases {
		header := http.Header{}

		if held.namespace != "" {
			header.Set(NamespaceHeader, held.namespace)
		}

		found := databaseFor(&incoming{host: held.host, header: header})

		if found != held.wanted {
			t.Errorf("%s: host %q with namespace %q gave %q, wanted %q",
				held.what, held.host, held.namespace, found, held.wanted)
		}
	}
}
