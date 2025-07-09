package validation

import (
	"encoding/json"
	"testing"

	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	guuid "github.com/google/uuid"
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

func TestCustomTypeValidation(t *testing.T) {
	type Input struct {
		ID   msuuid.UUID `validate:"required,uuid4" json:"id"`
		Type string      `validate:"required,mimetype" json:"type"`
	}

	tests := []struct {
		name       string
		in         Input
		wantErr    bool
		wantErrMap map[string]string
	}{
		{
			name:    "all good",
			in:      Input{ID: msuuid.UUID(guuid.New()), Type: "text/markdown"},
			wantErr: false,
		},
		{
			name:    "bad uuid, bad mimetype",
			in:      Input{ID: msuuid.UUID(guuid.Nil), Type: "application/x-foo"},
			wantErr: true,
			wantErrMap: map[string]string{
				"id":   "uuid4",
				"type": "mimetype",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateStruct() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				return
			}
			js, _ := ErrorsToJson(err)
			var got map[string]string
			if err := json.Unmarshal([]byte(js), &got); err != nil {
				t.Fatalf("json.Unmarshal err = %v", err)
			}
			for f, wantTag := range tt.wantErrMap {
				if got[f] != wantTag {
					t.Errorf("field %q: got %q, want %q", f, got[f], wantTag)
				}
			}
		})
	}
}

func TestNestedAndJsonTagFallback(t *testing.T) {
	type Inner struct {
		Foo string `validate:"required" json:"foo"`
	}
	type Outer struct {
		In  *Inner `validate:"required" json:"inner"`
		Bar int    `validate:"required"             `
	}

	// Case 1: nil pointer → error on "inner"
	t.Run("nil nested struct", func(t *testing.T) {
		o := Outer{In: nil, Bar: 0}

		err := ValidateStruct(o)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		js, _ := ErrorsToJson(err)

		var got map[string]string
		if err := json.Unmarshal([]byte(js), &got); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if got["inner"] != "required" {
			t.Errorf("inner: got %q, want %q", got["inner"], "required")
		}
		if got["Bar"] != "required" {
			t.Errorf("Bar: got %q, want %q", got["Bar"], "required")
		}
	})

	// Case 2: pointer present but Foo empty → error on "foo"
	t.Run("missing nested field", func(t *testing.T) {
		o := Outer{In: &Inner{Foo: ""}, Bar: 0}

		err := ValidateStruct(o)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		js, _ := ErrorsToJson(err)

		var got map[string]string
		if err := json.Unmarshal([]byte(js), &got); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		// Now the only failure on the nested struct is Foo → json:"foo"
		if got["foo"] != "required" {
			t.Errorf("foo: got %q, want %q", got["foo"], "required")
		}
		if got["Bar"] != "required" {
			t.Errorf("Bar: got %q, want %q", got["Bar"], "required")
		}
	})
}
