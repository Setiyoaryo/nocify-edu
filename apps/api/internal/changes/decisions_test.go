package changes

import "testing"

func TestValidateDecision(t *testing.T) {
	tests := []struct {
		status  string
		wantErr bool
	}{
		{StatusAcknowledged, false},
		{StatusDismissed, false},
		{StatusDeferred, false},
		{StatusPending, true}, // reviewers never set the implicit undecided state
		{"", true},
		{"maybe", true},
	}
	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			if err := ValidateDecision(tc.status); (err != nil) != tc.wantErr {
				t.Fatalf("ValidateDecision(%q) err = %v, wantErr %v", tc.status, err, tc.wantErr)
			}
		})
	}
}

func TestValidStatus(t *testing.T) {
	if !ValidStatus(StatusAcknowledged) || !ValidStatus(StatusDismissed) || !ValidStatus(StatusDeferred) {
		t.Fatal("ValidStatus() rejected a reviewer status")
	}
	if ValidStatus(StatusPending) || ValidStatus("other") {
		t.Fatal("ValidStatus() accepted a non-reviewer status")
	}
}
