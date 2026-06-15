package jobs

import "testing"

func TestValidateTarget(t *testing.T) {
	cases := []struct {
		kind, sid string
		labels    map[string]string
		wantErr   bool
	}{
		{"server", "s1", nil, false},
		{"server", "", nil, true},
		{"labels", "", map[string]string{"env": "prod"}, false},
		{"labels", "", map[string]string{}, true},
		{"labels", "", nil, true},
		{"", "s1", nil, true},
		{"bogus", "s1", nil, true},
	}
	for _, c := range cases {
		err := validateTarget(c.kind, c.sid, c.labels)
		if (err != nil) != c.wantErr {
			t.Errorf("kind=%q sid=%q labels=%v: err=%v wantErr=%v", c.kind, c.sid, c.labels, err, c.wantErr)
		}
	}
}
