package opts

import "testing"

// TestValidateAutoMemoryLimitRatio checks that the auto-memory-ratio flag is
// rejected outside (0, 1] while auto-memory-enabled is on, and ignored when it
// is off.
func TestValidateAutoMemoryLimitRatio(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		ratio   float64
		wantErr bool
	}{
		{name: "default", enabled: true, ratio: 0.9, wantErr: false},
		{name: "full limit", enabled: true, ratio: 1, wantErr: false},
		{name: "zero", enabled: true, ratio: 0, wantErr: true},
		{name: "negative", enabled: true, ratio: -0.5, wantErr: true},
		{name: "above one", enabled: true, ratio: 1.5, wantErr: true},
		{name: "disabled ignores ratio", enabled: false, ratio: 7, wantErr: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := &Options{AutoMemoryLimitEnabled: tc.enabled, AutoMemoryLimitRatio: tc.ratio}
			errs := o.validate()
			if tc.wantErr && len(errs) == 0 {
				t.Fatalf("expected a validation error for ratio %v", tc.ratio)
			}
			if !tc.wantErr && len(errs) != 0 {
				t.Fatalf("expected no validation error for ratio %v, got %v", tc.ratio, errs)
			}
		})
	}
}
