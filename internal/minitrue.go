package internal

func First[T any](val bool, a, b T) T {
	if val {
		return a
	}
	return b
}
