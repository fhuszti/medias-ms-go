package validation

import (
	"encoding/json"
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
