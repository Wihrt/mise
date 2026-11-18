package coverage

import "testing"

func TestErrorReturnsForms(t *testing.T) {
	cases := []struct {
		name, body string
		counted    bool
	}{
		{"plain", "if err != nil {\n\t\treturn err\n\t}", true},
		{"init", "if err := f(); err != nil {\n\t\treturn err\n\t}", true},
		{"else if", "if x {\n\t\tg()\n\t} else if err != nil {\n\t\treturn err\n\t}", true},
		{"suffix Err", "if closeErr != nil {\n\t\treturn closeErr\n\t}", true},
		{"suffix err", "if rerr != nil {\n\t\treturn rerr\n\t}", true},
		{"other condition", "if rc != 0 {\n\t\treturn rc\n\t}", false},
		{"statement before", "if err != nil {\n\t\tlog()\n\t\treturn err\n\t}", false},
		{"else body", "if x {\n\t\tg()\n\t} else {\n\t\treturn err\n\t}", false},
		{"equal nil", "if err == nil {\n\t\treturn err\n\t}", false},
		{"wrapped", "if err != nil {\n\t\treturn fmt.Errorf(\"x: %w\", err)\n\t}", true},
		{"swallowed", "if err != nil {\n\t\treturn nil\n\t}", false},
		{"bare return", "if err != nil {\n\t\treturn\n\t}", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ErrorReturns("package p\n\nfunc F() {\n\t" + c.body + "\n}\n")
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if c.counted {
				want = 1
			}
			if len(got) != want {
				t.Fatalf("got %v, want %d", got, want)
			}
		})
	}
}
