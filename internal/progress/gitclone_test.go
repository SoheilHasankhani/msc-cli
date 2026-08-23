package progress

import (
	"strings"
	"testing"
)

func TestParseCloneProgressReceivingObjects(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"Cloning into 'doctor-api'...",
		"remote: Enumerating objects: 1000, done.",
		"Receiving objects:  42% (420/1000), 1.20 MiB | 2.00 MiB/s",
		"this line is garbage and must be ignored",
		"Receiving objects: 100% (1000/1000), done.",
		"Resolving deltas:  50% (50/100)",
		"Resolving deltas: 100% (100/100), done.",
	}, "\n")

	var got []Update
	if err := ParseCloneProgress(strings.NewReader(input), "doctor-api", "doctor-api", func(u Update) {
		got = append(got, u)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("expected updates")
	}
	var last int64
	for _, u := range got {
		if u.Done {
			t.Fatalf("parser must not emit Done: %#v", u)
		}
		if u.Total != 100 {
			t.Fatalf("total = %d, want 100: %#v", u.Total, u)
		}
		if u.Current < last {
			t.Fatalf("bar went backwards: %d → %d", last, u.Current)
		}
		last = u.Current
	}
	if last < 90 {
		t.Fatalf("expected resolving to finish near 95, last=%d in %#v", last, got)
	}
}

func TestParseCloneProgressCarriageReturn(t *testing.T) {
	t.Parallel()

	input := "Receiving objects:  10% (10/100)\rReceiving objects:  40% (40/100)\rReceiving objects: 100% (100/100), done.\n"
	var got []Update
	if err := ParseCloneProgress(strings.NewReader(input), "r", "r", func(u Update) {
		got = append(got, u)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) < 3 {
		t.Fatalf("CR updates dropped: %#v", got)
	}
	if got[0].Current != 8 || got[1].Current != 32 || got[2].Current != 80 {
		t.Fatalf("mapped percents = %#v", got)
	}
}

func TestParseCloneProgressNeverFailsOnMalformed(t *testing.T) {
	t.Parallel()

	var n int
	if err := ParseCloneProgress(strings.NewReader("???\nnot progress\n"), "r", "r", func(Update) {
		n++
	}); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("emitted %d updates for garbage", n)
	}
}

func TestClonePhaseProgress(t *testing.T) {
	t.Parallel()
	cur := clonePhaseProgress("Receiving objects", 50)
	if cur != 40 {
		t.Fatalf("receiving 50%% = %d/100", cur)
	}
	cur = clonePhaseProgress("Resolving deltas", 0)
	if cur != 80 {
		t.Fatalf("resolving start = %d", cur)
	}
	cur = clonePhaseProgress("Checking out files", 100)
	if cur != 100 {
		t.Fatalf("checkout done = %d", cur)
	}
}
