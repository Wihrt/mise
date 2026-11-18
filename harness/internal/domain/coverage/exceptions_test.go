package coverage

import "testing"

func TestExceptionsRoundTripBackticks(t *testing.T) {
	text := "x := `raw`"
	f := Finding{Kind: "UNCOVERED", File: "a.go", Start: 3, End: 3, Text: text}
	md := Exceptions{}.WithDebt([]Finding{f}).Markdown()
	parsed, err := ParseExceptions(md)
	if err != nil {
		t.Fatal(err)
	}
	accepted, _, stale := parsed.Apply([]Finding{f})
	if len(accepted) != 1 || len(stale) != 0 {
		t.Errorf("markdown %q: accepted %v, stale %v; want the finding accepted", md, accepted, stale)
	}
}

func TestExceptionsOneEntryForSameTextTwice(t *testing.T) {
	a := Finding{Kind: "UNCOVERED", File: "a.go", Start: 3, End: 3, Text: "return 1"}
	b := Finding{Kind: "UNCOVERED", File: "a.go", Start: 9, End: 9, Text: "return 1"}
	e := Exceptions{}.WithDebt([]Finding{a, b})
	if n := len(e[0].Entries); n != 1 {
		t.Fatalf("baseline wrote %d entries, want 1", n)
	}
	accepted, refused, _ := e.Apply([]Finding{a, b})
	if len(accepted) != 2 || len(refused) != 0 {
		t.Errorf("accepted %d, refused %d; want both accepted", len(accepted), len(refused))
	}
}

func TestParseExceptionsErrors(t *testing.T) {
	for _, md := range []string{"- `x` : r\n", "## a.go\n\n- `x` :\n", "## a.go\n\n- fichier entier : \n"} {
		if _, err := ParseExceptions(md); err == nil {
			t.Errorf("%q: want an error", md)
		}
	}
}

func TestPragmaIsNeverAccepted(t *testing.T) {
	e := Exceptions{{File: "a.py", Entries: []Entry{{Whole: true, Reason: "r"}}}}
	accepted, _, _ := e.Apply([]Finding{{Kind: "PRAGMA", File: "a.py", Start: 1, End: 1}})
	if len(accepted) != 0 {
		t.Error("a pragma was accepted")
	}
}
