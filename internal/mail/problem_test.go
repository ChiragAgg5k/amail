package mail

import (
	"testing"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

func TestSettle(t *testing.T) {
	unanswered := Unanswered(DefaultTimeout)
	cases := []struct {
		name string
		err  error
		send bool
		hint string
	}{
		{"send timed out", unanswered, true, MaybeSent(unanswered).Hint},
		{"draft timed out", unanswered, false, unanswered.Hint},
		{"send refused", PermissionDenied(), true, PermissionDenied().Hint},
		{"send succeeded", nil, true, ""},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			settled := Settle(test.err, test.send)
			if failure.CodeOf(settled) != failure.CodeOf(test.err) || failure.HintOf(settled) != test.hint {
				t.Fatalf("settled %v (code %d, hint %q), want code %d, hint %q", settled, failure.CodeOf(settled), failure.HintOf(settled), failure.CodeOf(test.err), test.hint)
			}
		})
	}
	if settled := Settle(unanswered, true); settled.Error() != unanswered.Error() || failure.HintOf(settled) == unanswered.Hint {
		t.Fatalf("settled %q with hint %q", settled, failure.HintOf(settled))
	}
}
