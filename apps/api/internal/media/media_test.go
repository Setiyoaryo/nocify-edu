package media

import "testing"

func TestCheckTransition(t *testing.T) {
	lic := 1
	ok := AssetFacts{Status: "imported", Source: "DrawKit", Creator: "DrawKit", LicenseID: &lic, LicenseURL: "https://www.drawkit.com/license", Kind: "illustration", HasFile: true, SizeBytes: 120 << 10}
	steps := []string{"license_verified", "design_approved", "performance_checked", "published"}
	for _, to := range steps {
		if err := CheckTransition(ok, to); err != nil {
			t.Fatalf("%s → %s: %v", ok.Status, to, err)
		}
		ok.Status = to
	}

	if err := CheckTransition(AssetFacts{Status: "imported"}, "published"); err == nil {
		t.Error("skipping the lifecycle must fail")
	}
	if err := CheckTransition(AssetFacts{Status: "imported", Source: "x", Creator: "y"}, "license_verified"); err == nil {
		t.Error("missing license must fail")
	}
	if err := CheckTransition(AssetFacts{Status: "license_verified", AIGenerated: true}, "design_approved"); err == nil {
		t.Error("AI-generated imagery must not be approved")
	}
	big := AssetFacts{Status: "design_approved", Kind: "icon", HasFile: true, SizeBytes: 80 << 10}
	if err := CheckTransition(big, "performance_checked"); err == nil {
		t.Error("over-budget file must fail performance check")
	}
	if err := CheckTransition(AssetFacts{Status: "design_approved", Kind: "icon"}, "performance_checked"); err == nil {
		t.Error("missing file must fail performance check")
	}
	if err := CheckTransition(AssetFacts{Status: "published"}, "rejected"); err != nil {
		t.Error("anything can be rejected")
	}
}
