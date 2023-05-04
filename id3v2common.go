package tag

// Identical in ID3v2[234].
type ID3v2Frame struct {
	Key   string
	Value []byte
}

func getStringImpl(name string, frames []ID3v2Frame) (string, error) {
	for _, f := range frames {
		if f.Key == name {
			return GetString(f.Value)
		}
	}
	return "", ErrTagNotFound
}

