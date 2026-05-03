package validation

import (
	"github.com/go-playground/validator/v10"
)

type Validator struct {
	v *validator.Validate
}

func New() *Validator {
	v := validator.New()

	registerCustom(v)

	return &Validator{v: v}
}

func (v *Validator) Struct(s any) error {
	return v.v.Struct(s)
}
