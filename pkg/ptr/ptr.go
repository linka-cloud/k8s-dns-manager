package ptr

func Bool(b bool) *bool {
	return &b
}

func ToBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func ToBoolD(b *bool, d bool) bool {
	if b == nil {
		return d
	}
	return *b
}
