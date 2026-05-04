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

// Struct validates explicit service/CLI structs outside Fiber binding.
func (v *Validator) Struct(s any) error {
	return v.v.Struct(s)
}

// Validate satisfies Fiber's StructValidator interface used by c.Bind().Body(...).
func (v *Validator) Validate(out any) error {
	return v.Struct(out)
}
