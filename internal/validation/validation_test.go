package validation

import (
	"encoding/json"
	"testing"
)

func TestValidateStructAndErrorsToJson(t *testing.T) {
	type Input struct {
		Email string `validate:"required,email"  json:"email"`
		Tags  []int  `validate:"min=1,dive,gt=0" json:"tags"`
	}

	tests := []struct {
		name        string
		in          Input
		wantErr     bool
		wantJsonMap map[string]string
	}{
		{
			name:    "success",
			in:      Input{Email: "a@b.com", Tags: []int{1, 2, 3}},
			wantErr: false,
		},
		{
			name:    "missing email",
			in:      Input{Email: "", Tags: []int{1}},
			wantErr: true,
			wantJsonMap: map[string]string{
				"email": "required",
			},
		},
		{
			name:    "invalid email and empty tags",
			in:      Input{Email: "not-an-email", Tags: []int{}},
			wantErr: true,
			wantJsonMap: map[string]string{
				"email": "email",
				"tags":  "min",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				return
			}

			// convert and unmarshal for comparison
			js, jerr := ErrorsToJson(err)
			if jerr != nil {
				t.Fatalf("ErrorsToJson() error = %v", jerr)
			}
			var got map[string]string
			if err := json.Unmarshal([]byte(js), &got); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			for field, tag := range tt.wantJsonMap {
				if got[field] != tag {
					t.Errorf("field %q: got %q, want %q", field, got[field], tag)
				}
			}
		})
	}
}
