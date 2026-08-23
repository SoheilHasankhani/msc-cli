package complete

import "testing"

func TestComposeTakesServices(t *testing.T) {
	t.Parallel()
	for _, sub := range []string{"logs", "exec", "up", "restart"} {
		if !ComposeTakesServices(sub) {
			t.Fatalf("%q should take services", sub)
		}
	}
	for _, sub := range []string{"config", "ps", "images", "ls"} {
		if ComposeTakesServices(sub) {
			t.Fatalf("%q should not take services", sub)
		}
	}
}

func TestComposeSubcommandsSorted(t *testing.T) {
	t.Parallel()
	got := ComposeSubcommands()
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Fatalf("not sorted: %v", got)
		}
	}
}
