package progress

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestTeaModelAppliesUpdateAndViewsLabel(t *testing.T) {
	t.Parallel()

	m := newTeaModel(80, nil)
	next, cmd := m.Update(updateMsg{ID: "doctor", Label: "doctor", Current: 1, Total: 4})
	tm, ok := next.(teaModel)
	if !ok {
		t.Fatalf("%T", next)
	}
	if tm.inner.Bars["doctor"].Percent != 0.25 {
		t.Fatalf("percent = %v", tm.inner.Bars["doctor"].Percent)
	}
	if cmd == nil {
		t.Fatal("SetPercent should return an animation cmd")
	}
	if !strings.Contains(tm.View(), "doctor") {
		t.Fatalf("view = %q", tm.View())
	}
}

func TestTeaModelQuitOnDone(t *testing.T) {
	t.Parallel()

	m := newTeaModel(80, nil)
	next, cmd := m.Update(quitMsg{})
	tm := next.(teaModel)
	if !tm.quitting {
		t.Fatal("expected quit")
	}
	if cmd == nil {
		t.Fatal("expected tea.Quit")
	}
}

func TestFuncSourceEmits(t *testing.T) {
	t.Parallel()

	src := FuncSource{
		ID:    "identity-api",
		Label: "identity-api",
		Fn: func(ctx context.Context, emit func(Update)) error {
			emit(Update{Current: 0, Total: 1})
			emit(Update{Current: 1, Total: 1, Done: true})
			return nil
		},
	}
	ch := make(chan Update, 4)
	if err := src.Run(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	close(ch)
	var got []Update
	for u := range ch {
		got = append(got, u)
	}
	if len(got) != 2 || got[0].ID != "identity-api" || !got[1].Done {
		t.Fatalf("%#v", got)
	}
}

func TestRunTTYCompletesWithoutFallbackLines(t *testing.T) {
	t.Parallel()

	src := scriptedSource{id: "alpine", events: []Update{
		{ID: "alpine", Label: "alpine", Current: 1, Total: 2},
		{ID: "alpine", Label: "alpine", Current: 2, Total: 2, Done: true},
	}}
	tty := true
	if err := Run(context.Background(), []Source{src}, Options{Output: io.Discard, IsTTY: &tty}); err != nil {
		t.Fatal(err)
	}
}
