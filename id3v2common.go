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

func setStringImpl(name, value  string, frames []ID3v2Frame) []ID3v2Frame {
	newframe := ID3v2Frame{
		Key:   name,
		Value: SetString(value),
	}

	for _, f := range frames {
		if f.Key == name {
			f.Value = newframe.Value
			return frames
		}
	}

	return  append(frames, newframe)
}
