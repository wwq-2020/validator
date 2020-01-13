package t

import (
	"errors"
)

// ValidateU Validate U
func ValidateU(src U) error {

	if src.Name == "" {
		return errors.New("field Name can't be empty")
	}

	if src.Value == 0 {
		return errors.New("field Value can't be empty")
	}

	return nil
}

// ValidateUPointer Validate UPointer
func ValidateUPointer(src *U) error {
	if src == nil {
		return errors.New("src can't be nil")
	}

	if src.Name == "" {
		return errors.New("field Name can't be empty")
	}

	if src.Value == 0 {
		return errors.New("field Value can't be empty")
	}

	return nil
}
