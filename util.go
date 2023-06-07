package tag

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"io"
	"net/http"
	"unicode/utf16"
	"unicode/utf8"
	"strconv"
	"strings"
	"os"

	"log"
)

func seekAndRead(input io.ReadSeeker, offset int64, whence int, read int) ([]byte, error) {
	if input == nil {
		return nil, ErrEmptyFile
	}

	_, err := input.Seek(offset, whence)
	if err != nil {
		return nil, ErrSeekFile
	}

	data := make([]byte, read)
	nReaded, err := input.Read(data)
	if err != nil {
		return nil, err
	}
	if nReaded != read {
		return nil, ErrReadFile
	}

	return data, nil
}

func seekAndReadString(input io.ReadSeeker, offset int64, whence int, read int) (string, error) {
	data, err := seekAndRead(input, offset, whence, read)
	return string(data), err
}

func readBytes(input io.Reader, size int) ([]byte, error) {
	if input == nil {
		return nil, ErrEmptyFile
	}

	data := make([]byte, size)
	nReaded, err := input.Read(data)
	if err != nil {
		return nil, err
	}

	if nReaded != size {
		return nil, ErrReadFile
	}

	return data, nil
}

// TODO(rjk): This is not standard compliant? As I understand the Go spec,
// ASCII >= 128 will not be correclty encoded.
func GetEncoding(code byte) string {
	if code == 0 || code == 3 {
		return encodingUTF8
	}
	if code == 1 {
		return encodingUTF16
	}
	if code == 2 {
		return encodingUTF16BE
	}
	return ""
}

// TextEncoding -
// Text Encoding for text frame header
// First byte determinate text encoding.
// If ISO-8859-1 is used this byte should be $00, if Unicode is used it should be $01
// Return text encoding. E.g. "utf8", "utf16", etc.
func TextEncoding(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	return GetEncoding(b[0])
}

// TODO(rjk): This should truncate the string?
func DecodeString(b []byte, encoding string) (string, error) {
	switch encoding {
	case encodingUTF8:
		return string(b), nil
	case encodingUTF16:
		value, err := DecodeUTF16(b)
		if err != nil {
			return "", err
		}
		return value, nil
	case encodingUTF16BE:
		return DecodeUTF16BE(b)
	}

	return "", ErrEncodingFormat
}

// Decode UTF-16 Little Endian to UTF-8.
// TODO(rjk): Might consider doing this lazily?
func DecodeUTF16(b []byte) (string, error) {
	if len(b)%2 != 0 {
		return "", ErrDecodeEvenLength
	}

	u16s := make([]uint16, 1)

	ret := &bytes.Buffer{}

	b8buf := make([]byte, 4)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[0] = uint16(b[i]) + (uint16(b[i+1]) << 8)
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}

	return ret.String(), nil
}

// Decode UTF-16 Big Endian To UTF-8.
func DecodeUTF16BE(b []byte) (string, error) {
	if len(b)%2 != 0 {
		return "", ErrDecodeEvenLength
	}

	u16s := make([]uint16, 1)

	ret := &bytes.Buffer{}

	b8buf := make([]byte, 4)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[0] = uint16(b[i+1]) + (uint16(b[i]) << 8)
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}

	return ret.String(), nil
}

// ByteToIntSynchsafe -
// Convert byte to int
// In some parts of the tag it is inconvenient to use the
// unsychronisation scheme because the size of unsynchronised data is
// not known in advance, which is particularly problematic with size
// descriptors. The solution in ID3v2 is to use synchsafe integers, in
// which there can never be any false synchs. Synchsafe integers are
// integers that keep its highest bit (bit 7) zeroed, making seven bits
// out of eight available. Thus a 32 bit synchsafe integer can store 28
// bits of information.
func ByteToIntSynchsafe(data []byte) int {
	result := 0
	for _, b := range data {
		result = (result << 7) | int(b)
	}
	return result
}

func IntToByteSynchsafe(data int) []byte {
	// 7F = 0111 1111
	return []byte{
		byte(data>>23) & 0x7F,
		byte(data>>15) & 0x7F,
		byte(data>>7) & 0x7F,
		byte(data) & 0x7F,
	}
}

// Convert byte to int.
func ByteToInt(data []byte) int {
	result := 0
	for _, b := range data {
		result = (result << 8) | int(b)
	}
	return result
}

// Return bit value
// Index starts from 0
// bits order [7,6,5,4,3,2,1,0].
func GetBit(data byte, index byte) byte {
	return 1 & (data >> index)
}

func SetBit(data *byte, bit bool, index byte) {
	if bit {
		*data |= 1 << index
	} else {
		*data &= ^(1 << index)
	}
}

// TODO(rjk): This might be mis-named.
func GetString(b []byte, err error) (string, error) {
	if err != nil {
		return "", err
	}

	if len(b) < 2 {
		return "", ErrIncorrectTag
	}
	return DecodeString(b[1:], TextEncoding(b))
}

func SetString(value string) []byte {
	// Set UTF-8
	result := []byte{0}
	// Set data
	return append(result, []byte(value)...)
}

// Read format:
// [length, data]
// length in littleEndian.
func readLengthData(input io.Reader, order binary.ByteOrder) ([]byte, error) {
	// length
	var length uint32
	err := binary.Read(input, order, &length)
	if err != nil {
		return nil, err
	}

	// data
	data, err := readBytes(input, int(length))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func writeLengthData(output io.Writer, order binary.ByteOrder, data []byte) error {
	length := uint32(len(data))
	err := binary.Write(output, order, length)
	if err != nil {
		return err
	}

	_, err = output.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func downloadImage(url string) (image.Image, error) {
	// nolint:gosec
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	return img, err
}

// nolint:deadcode,unused
func colorModelToBitsPerPixel(model color.Model) int {
	var bpp int
	switch model {
	case color.RGBAModel:
		bpp = 8
	case color.RGBA64Model:
		bpp = 64
	case color.NRGBAModel:
		bpp = 8
	case color.NRGBA64Model:
		bpp = 64
	case color.AlphaModel:
		bpp = 8
	case color.Alpha16Model:
		bpp = 16
	case color.GrayModel:
		bpp = 8
	case color.Gray16Model:
		bpp = 16
	default:
		bpp = 8
	}

	return bpp
}

func SplitBytesWithTextDescription(data []byte, encoding string) [][]byte {
	separator := []byte{0}
	if encoding == encodingUTF16 || encoding == encodingUTF16BE {
		separator = []byte{0, 0}
	}

	result := bytes.SplitN(data, separator, 2)
	if len(result) != 2 {
		return result
	}

	if len(result[1]) == 0 {
		return result
	}

	if result[1][0] == 0 {
		result[0] = append(result[0], result[1][0])
		result[1] = result[1][1:]
	}
	return result
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

func sizePackID3v234(length int) []byte {
	header := make([]byte, id3v24FrameHeaderSize)
		header[4] = byte(length >> 24)
		header[5] = byte(length >> 16)
		header[6] = byte(length >> 8)
		header[7] = byte(length)
	return header
}

func writeFramesImpl(writer io.Writer, hf map[string][]byte, packer func(int)[]byte) error {
	for k, v := range hf {

log.Printf("%s: length %d", k, len(v))

		header := packer(len(v))

		// Frame id
		copy(header, k)

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
	}
	return nil
}

func getFramesLength(f map[string][]byte, headersz int) int {
	result := 0
	// TODO(rjk): Make the size of the header configurable.
	for _, v := range f {
		// headersz - size of tag header
		result += headersz + len(v)
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

type ID3v2 struct {
	Marker     string // Always 'ID3'
	Version    Version
	SubVersion int
	Flags      id3v2Flags
	Length     int
	Frames     map[string][]byte
	UserFrames map[string][]byte

	Data []byte
}

// Identical in ID3v2[234].
type ID3v2Frame struct {
	Key   string
	Value []byte
}

type id3v2Flags byte

func (flags id3v2Flags) String() string {
	return strconv.Itoa(int(flags))
}

func (flags id3v2Flags) IsUnsynchronisation() bool {
	return GetBit(byte(flags), 7) == 1
}

func (flags id3v2Flags) SetUnsynchronisation(data bool) {
	SetBit((*byte)(&flags), data, 7)
}

func (flags id3v2Flags) HasExtendedHeader() bool {
	return GetBit(byte(flags), 6) == 1
}

func (flags id3v2Flags) SetExtendedHeader(data bool) {
	SetBit((*byte)(&flags), data, 7)
}

func (flags id3v2Flags) IsExperimentalIndicator() bool {
	return GetBit(byte(flags), 5) == 1
}

func (flags id3v2Flags) SetExperimentalIndicator(data bool) {
	SetBit((*byte)(&flags), data, 7)
}

// getSplitNumberImpl returns the 3 cases 
// There are 3 cases: num, num / num or error.
func getSplitNumberImpl(tname string, frames map[string][]byte) (int, int, error) {
	disknumbers, err := getStringImpl(tname, frames)
	if err != nil {
		return 0, 0, err
	}

	numbers := strings.Split(disknumbers, "/")
	if len(numbers)  < 1 {
		return 0, 0, ErrIncorrectLength
	}
	number, err := strconv.Atoi(numbers[0])
	if err != nil {
		return 0, 0, err
	}

	if len(numbers) == 1 {
		return number, number, nil
	}
	if len(numbers) > 2 {
		return 0, 0, ErrIncorrectLength
	}
	total, err := strconv.Atoi(numbers[1])
	if err != nil {
		return number, 0, err
	}
	return number, total, nil
}

func setAttachedPictureImpl(tname string, picture *AttachedPicture, frames map[string][]byte)  {
	// set UTF-8
	result := []byte{0}

	// MIME type
	result = append(result, []byte(picture.MIME)...)
	result = append(result, 0x00)

	// Picture type
	result = append(result, picture.PictureType)

	// Picture description
	result = append(result, []byte(picture.Description)...)
	result = append(result, 0x00)

	// Picture data
	result = append(result, picture.Data...)

// TODO(rjk): The string conversion seems suprfluous.
// I have discarded the 
	frames[tname] = result
}

// TODO(rjk): Use buffered I/O
func saveFileImpl(sm SaveMetadata, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return sm.Save(file)
}

type AllHeaderFields [][]byte

// size is the total size of all of the header fields.
func (hfs AllHeaderFields) size() int {
	sz := 0
	for _, h := range hfs {
		sz += len(h)
	}
	return sz
}		

// writeHeaderImpl writes an ID3v2 header payload in a configurable
// fashion to writer. hf defines an of header fields with their
// associated size in bytes.
func writeHeaderImpl(writer io.Writer,  hf AllHeaderFields) error {
	headerByte := make([]byte, hf.size())

	start := 0
	for _, h := range hf {
		copy(headerByte[start:start+len(h)], h)
		start += len(h)
	}

	nWriten, err := writer.Write(headerByte)
	if err != nil {
		return err
	}
	if nWriten != len(headerByte) {
		return ErrWriting
	}
	return nil

}
