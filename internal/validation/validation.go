package validation

import (
	"encoding/json"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/google/uuid"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())

	// Tell the validator to use the JSON tag as the “field name”
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		// Grab the value of `json:"foo,omitempty"`
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			// fallback to the Go field name or skip
			return fld.Name
		}
		return name
	})

	// Validate db.UUID as string
	validate.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
		if v, ok := field.Interface().(db.UUID); ok {
			u := uuid.UUID(v)
			return u.String()
		}
		return nil
	}, db.UUID{})

	// Validate mime types
	err := validate.RegisterValidation("mimetype", func(fl validator.FieldLevel) bool {
		v := fl.Field().String()
		return media.IsMimeTypeAllowed(v)
	})
	if err != nil {
		return
	}
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

func ErrorsToJson(validationErrs error) (string, error) {
	errsMap := make(map[string]string)
	for _, fieldErr := range validationErrs.(validator.ValidationErrors) {
		errsMap[fieldErr.Field()] = fieldErr.Tag()
	}

	errsJson, err := json.Marshal(errsMap)
	if err != nil {
		return "", err
	}
	return string(errsJson), nil
}
