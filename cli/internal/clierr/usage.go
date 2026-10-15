package clierr

type UsageError struct {
	Msg string
}

func (e *UsageError) Error() string {
	return e.Msg
}

func Usage(msg string) error {
	return &UsageError{Msg: msg}
}
