package tag

import (
	"io"
	"strconv"
	"strings"
)

// Identical in ID3v2[234].
type ID3v2Frame struct {
	Key   string
	Value []byte
}

func getBytesImpl(k string, frames map[string][]byte) ([]byte, error) {
	v, ok := frames[k]
	if !ok {
		return []byte{}, ErrTagNotFound
	}
	return v, nil
}

func getStringImpl(k string, frames map[string][]byte) (string, error) {
	return GetString(getBytesImpl(k, frames))
}

func setStringImpl(k, v string, frames map[string][]byte) {
	frames[k] = SetString(v)
}

func wrappedAtoi(str string, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str)
}

func getAllTagNamesImpl(f, uf map[string][]byte) []string {
	names := make([]string, 0, len(f)+len(uf))
	for k := range f {
		names = append(names, k)
	}
	for k := range uf {
		names = append(names, k)
	}
	return names
}

func writeFramesImpl(writer io.Writer, frames, userframes map[string][]byte) error {
	for k, v := range frames {
		header := make([]byte, 10)

		// Frame id
		copy(header, k)

		// Frame size
		length := len(v)
		header[4] = byte(length >> 24)
		header[5] = byte(length >> 16)
		header[6] = byte(length >> 8)
		header[7] = byte(length)

		// write header
		_, err := writer.Write(header)
		if err != nil {
			return err
		}

		// write data
		_, err = writer.Write(v)
		if err != nil {
			return err
		}

		// TODO(rjk): Write user frames
	}
	return nil
}

func getFramesLength(f map[string][]byte) int {
	result := 0
	// TODO(rjk): Make the size of the header configurable.
	for _, v := range f {
		// 10 - size of tag header
		result += 10 + len(v)
	}
	return result
}

// TODO(rjk): This does way more work than necessary.
func splitUserFrameValue(v []byte) (string, []byte, error) {
	// First byte in v is the text encoding.

	str, err := GetString(v, nil)
	if err != nil {
		return "", []byte{}, err
	}
	info := strings.SplitN(str, "\x00", 2)
	if len(info) != 2 {
		return "", []byte{}, ErrIncorrectTag
	}

	val := make([]byte, 1 /* encoding */ +len(info[1] /* value  */))
	val[0] = v[0]
	copy(val[1:], info[1])

	return info[0], val, nil
}
