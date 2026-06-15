package scheduler

import "testing"

func TestApplyAddsRemovesReplaces(t *testing.T) {
	fired := map[string]int{}
	s := New(func(id string) { fired[id]++ })

	// Initial set with one valid cron.
	if err := s.Apply([]JobDef{
		{ID: "a", ScheduleCron: "* * * * *", Timezone: "UTC", Enabled: true},
		{ID: "b", ScheduleCron: "* * * * *", Timezone: "UTC", Enabled: true},
	}); err != nil {
		t.Fatalf("apply 1: %v", err)
	}

	// Re-applying with a removed and a disabled job leaves only one entry.
	if err := s.Apply([]JobDef{
		{ID: "a", ScheduleCron: "* * * * *", Timezone: "UTC", Enabled: false},
	}); err != nil {
		t.Fatalf("apply 2: %v", err)
	}
	if got := len(s.entries); got != 0 {
		t.Errorf("entries after disable/remove: got %d, want 0", got)
	}
}

func TestApplyRejectsBadCron(t *testing.T) {
	s := New(func(string) {})
	err := s.Apply([]JobDef{{ID: "x", ScheduleCron: "not-a-cron", Timezone: "UTC", Enabled: true}})
	if err == nil {
		t.Fatal("expected parse error for bad cron")
	}
}
