package file

import "testing"

func TestResolveConfinesToRoot(t *testing.T) {
	c := &connection{root: "/projects/cleanfit/rill"}
	bad := []string{
		"../../adirena/rill/.env",
		"../adirena/rill/.env",
		"/etc/passwd/../../../etc/passwd",
		"sources/../../../adirena/rill/.env",
	}
	for _, p := range bad {
		if got, err := c.resolve(p); err == nil {
			t.Errorf("ESCAPED: %q -> %q", p, got)
		}
	}
	good := []string{"rill.yaml", "sources/dim_orders.yaml", "./metrics/x.yaml", "a/../rill.yaml"}
	for _, p := range good {
		if _, err := c.resolve(p); err != nil {
			t.Errorf("wrongly blocked: %q -> %v", p, err)
		}
	}
}
