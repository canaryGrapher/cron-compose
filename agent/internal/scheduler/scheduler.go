// Package scheduler runs a local cron loop over the job definitions the agent holds.
// It is timezone-aware (per job) and emits a callback when a job fires.
package scheduler

import (
	"sync"

	"github.com/robfig/cron/v3"
)

// JobDef is the slim view of a job the scheduler cares about. The full job spec lives
// in the local store and is passed back to the fire callback.
type JobDef struct {
	ID           string
	ScheduleCron string
	Timezone     string
	Enabled      bool
}

// FireFunc is called when a job's schedule triggers.
type FireFunc func(jobID string)

// Scheduler manages a cron.Cron instance per timezone, so DST is handled correctly.
type Scheduler struct {
	mu      sync.Mutex
	crons   map[string]*cron.Cron // tz -> cron
	entries map[string]entry      // jobID -> entry
	fire    FireFunc
}

type entry struct {
	tz string
	id cron.EntryID
}

// New returns an empty scheduler. Call Apply to add/update jobs, then Start.
func New(fire FireFunc) *Scheduler {
	return &Scheduler{
		crons:   map[string]*cron.Cron{},
		entries: map[string]entry{},
		fire:    fire,
	}
}

// Apply syncs the scheduler to the given job set: adds new ones, replaces changed ones,
// removes any not present. Callers should hold the full intended set, not a delta.
func (s *Scheduler) Apply(jobs []JobDef) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	want := map[string]JobDef{}
	for _, j := range jobs {
		want[j.ID] = j
	}

	// remove jobs no longer present
	for id, e := range s.entries {
		if _, ok := want[id]; !ok {
			s.crons[e.tz].Remove(e.id)
			delete(s.entries, id)
		}
	}

	// add or replace
	for id, j := range want {
		if existing, ok := s.entries[id]; ok {
			s.crons[existing.tz].Remove(existing.id)
			delete(s.entries, id)
		}
		if !j.Enabled {
			continue
		}
		c, err := s.cronFor(j.Timezone)
		if err != nil {
			return err
		}
		jobID := id
		entryID, err := c.AddFunc(j.ScheduleCron, func() { s.fire(jobID) })
		if err != nil {
			return err
		}
		s.entries[id] = entry{tz: j.Timezone, id: entryID}
	}
	return nil
}

// Start kicks every per-tz cron loop.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.crons {
		c.Start()
	}
}

// Stop halts every per-tz cron loop.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.crons {
		c.Stop()
	}
}

func (s *Scheduler) cronFor(tz string) (*cron.Cron, error) {
	if c, ok := s.crons[tz]; ok {
		return c, nil
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	c := cron.New(cron.WithParser(parser), cron.WithLocation(loadLocation(tz)))
	s.crons[tz] = c
	return c, nil
}
