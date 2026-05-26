package errors

import "fmt"

func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

func Wrapf(op string, err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	detail := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s(%s): %w", op, detail, err)
}
