package bratrax

import "testing"

func TestRedactDSN(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"keyword/value (the production form)",
			"host=localhost port=5435 user=u password=SUPERSECRET dbname=bratrax sslmode=verify-ca",
			"host=localhost port=5435 user=u password=[redacted] dbname=bratrax sslmode=verify-ca"},
		{"password last field",
			"host=h dbname=d password=SUPERSECRET",
			"host=h dbname=d password=[redacted]"},
		{"single quoted with spaces",
			"host=h password='se cr et' dbname=d",
			"host=h password=[redacted] dbname=d"},
		{"sslpassword too",
			"host=h sslpassword=SUPERSECRET dbname=d",
			"host=h sslpassword=[redacted] dbname=d"},
		{"uri form",
			"postgres://u:SUPERSECRET@localhost:5435/bratrax?sslmode=verify-ca",
			"postgres://localhost:5435/bratrax?sslmode=verify-ca"},
		{"no password at all",
			"host=h dbname=d sslmode=verify-ca",
			"host=h dbname=d sslmode=verify-ca"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := redactDSN(c.in)
			if got != c.want {
				t.Errorf("\n in:   %s\n got:  %s\n want: %s", c.in, got, c.want)
			}
			if got == c.in && c.in != c.want {
				t.Errorf("redaction was a NO-OP for %q", c.in)
			}
		})
	}
}
