package planmodel

import "testing"

func TestAttributePath_String(t *testing.T) {
	idx0 := 0
	cases := []struct {
		path AttributePath
		want string
	}{
		{AttributePath{{Key: "metadata"}, {Key: "labels"}, {Key: "app.kubernetes.io/version"}}, `metadata.labels["app.kubernetes.io/version"]`},
		{AttributePath{{Key: "ingress"}, {Key: "rule"}, {Index: &idx0}, {Key: "host"}}, "ingress.rule[0].host"},
		{AttributePath{{Key: "id"}}, "id"},
	}
	for _, c := range cases {
		if got := c.path.String(); got != c.want {
			t.Errorf("String() = %q, want %q", got, c.want)
		}
	}
}

func TestAttributePath_HasSuffix(t *testing.T) {
	idx0, idx1 := 0, 1
	full := AttributePath{{Key: "metadata"}, {Key: "labels"}, {Key: "app.kubernetes.io/version"}}
	suffix := AttributePath{{Key: "labels"}, {Key: "app.kubernetes.io/version"}}
	if !full.HasSuffix(suffix) {
		t.Error("expected full to have suffix")
	}
	if full.HasSuffix(AttributePath{{Key: "labels"}, {Key: "team"}}) {
		t.Error("did not expect a differing key to match")
	}
	if (AttributePath{{Index: &idx0}}).HasSuffix(AttributePath{{Index: &idx1}}) {
		t.Error("differing indices should not match")
	}
	if suffix.HasSuffix(full) {
		t.Error("a longer suffix than the path itself must not match")
	}
}

func TestAttributePath_HasPrefix(t *testing.T) {
	idx0, idx1 := 0, 1
	full := AttributePath{{Key: "triggers"}, {Key: "app.kubernetes.io/version"}}
	prefix := AttributePath{{Key: "triggers"}}

	if !full.HasPrefix(prefix) {
		t.Error("expected full to have prefix")
	}
	if !full.HasPrefix(full) {
		t.Error("a path must be its own prefix (exact match)")
	}
	if full.HasPrefix(AttributePath{{Key: "spec"}}) {
		t.Error("did not expect a differing key to match")
	}
	if (AttributePath{{Index: &idx0}}).HasPrefix(AttributePath{{Index: &idx1}}) {
		t.Error("differing indices should not match")
	}
	if prefix.HasPrefix(full) {
		t.Error("a longer prefix than the path itself must not match")
	}
}
