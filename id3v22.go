package tag

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"regexp"
	"strconv"
	"time"
	"fmt"
)

//type ID3v22Frame = ID3v2Frame

type ID3v22 struct {
	ID3v2
}

func (id3v2 *ID3v22) GetAllTagNames() []string {
	return getAllTagNamesImpl(id3v2.Frames, id3v2.UserFrames)
}

func (id3v2 *ID3v22) GetVersion() Version {
	return VersionID3v22
}

func (id3v2 *ID3v22) GetFileData() []byte {
	return id3v2.Data
}

func (id3v2 *ID3v22) GetTitle() (string, error) {
	return id3v2.GetString("TT2")
}

func (id3v2 *ID3v22) GetArtist() (string, error) {
	return id3v2.GetString("TP1")
}

func (id3v2 *ID3v22) GetAlbum() (string, error) {
	return id3v2.GetString("TAL")
}

func (id3v2 *ID3v22) GetYear() (int, error) {
	return wrappedAtoi(getStringImpl("TYE", id3v2.Frames))
}

// TODO(rjk): This is wrong in the same way as the ID3v24 version.
// Also: I am not correctly handling the situation with multiple comments.
// TODO(rjk): Add support for multiple comments.
func (id3v2 *ID3v22) GetComment() (string, error) {
	commentStr, err := getStringImpl("COM", id3v2.Frames)
	if err != nil {
		return "", err
	}
	if len(commentStr) < 4 {
		return "", ErrIncorrectLength
	}
	return commentStr[4:], nil
}

// TODO(rjk): Is there commonality with this?
func (id3v2 *ID3v22) GetGenre() (string, error) {
	genre, err := id3v2.GetString("TCO")
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`\([0-9]+\)`)
	match := re.FindAllString(genre, -1)
	if len(match) > 0 {
		code, err := strconv.Atoi(match[0][1 : len(match[0])-1])
		if err != nil {
			return "", err
		}
		return genres[Genre(code)], nil
	}
	return "", nil
}

func (id3v2 *ID3v22) GetAlbumArtist() (string, error) {
	return "", fmt.Errorf("GetAlbumArtist not available in ID3v22")
}

// TODO(rjk): time management has changed a lot between the tag versions.
// Only month and day are available from TDA. Figure out how to proceed
// here.
func (id3v2 *ID3v22) GetDate() (time.Time, error) {
	panic("implement me")

	// "TDA"
	// date, err := id3v2.GetString("TDA")
	//if err != nil {
	//	return "", err
	//}
}

func (id3v2 *ID3v22) GetArranger() (string, error) {
	return getStringImpl("TP4", id3v2.Frames)
}

func (id3v2 *ID3v22) GetAuthor() (string, error) {
	return getStringImpl("TOL", id3v2.Frames)
}

func (id3v2 *ID3v22) GetBPM() (int, error) {
	return wrappedAtoi(getStringImpl("TBP", id3v2.Frames))
}

func (id3v2 *ID3v22) GetCatalogNumber() (string, error) {
	// This is non-standard.
	return id3v2.GetStringTXX("CATALOGNUMBER")
}

func (id3v2 *ID3v22) GetCompilation() (string, error) {
	return "", fmt.Errorf("GetCompilation not available in ID3v22")
}

func (id3v2 *ID3v22) GetComposer() (string, error) {
	return getStringImpl("TCM", id3v2.Frames)
}

func (id3v2 *ID3v22) GetConductor() (string, error) {
	return getStringImpl("TP3", id3v2.Frames)
}

func (id3v2 *ID3v22) GetCopyright() (string, error) {
	return getStringImpl("TCR", id3v2.Frames)
}

func (id3v2 *ID3v22) GetDescription() (string, error) {
	return getStringImpl("TT3", id3v2.Frames)
}

func (id3v2 *ID3v22) GetDiscNumber() (int, int, error) {
	return getSplitNumberImpl("TPA", id3v2.Frames )
}

func (id3v2 *ID3v22) GetEncodedBy() (string, error) {
	return id3v2.GetString("TEN")
}

func (id3v2 *ID3v22) GetTrackNumber() (int, int, error) {
	return getSplitNumberImpl("TRK", id3v2.Frames )
}

func (id3v2 *ID3v22) GetPicture() (image.Image, error) {
	pic, err := id3v2.GetAttachedPicture()
	if err != nil {
		return nil, err
	}
	switch pic.MIME {
	case mimeImageJPEG:
		return jpeg.Decode(bytes.NewReader(pic.Data))
	case mimeImagePNG:
		return png.Decode(bytes.NewReader(pic.Data))
	default:
		return nil, ErrIncorrectTag
	}
}

func (id3v2 *ID3v22) GetAttachedPicture() (*AttachedPicture, error) {
	var picture AttachedPicture

	bytes, err := id3v2.GetBytes("PIC")
	if err != nil {
		return nil, err
	}

	textEncoding := bytes[0]
	mimeText := string(bytes[1:4])
	if mimeText == "JPG" {
		picture.MIME = mimeImageJPEG
	} else if mimeText == "PNG" {
		picture.MIME = mimeImagePNG
	}

	picture.PictureType = bytes[4]

	values := SplitBytesWithTextDescription(bytes[5:], GetEncoding(textEncoding))
	if len(values) != 2 {
		return nil, ErrIncorrectTag
	}

	desc, err := DecodeString(values[0], GetEncoding(textEncoding))
	if err != nil {
		return nil, err
	}

	picture.Description = desc
	picture.Data = values[1]

	return &picture, nil
}

// GetStringTXX - get user frame
func (id3v2 *ID3v22) GetStringTXX(name string) (string, error) {
	return getStringImpl(name, id3v2.UserFrames)
}

func (id3v2 *ID3v22) GetIntTXX(name string) (int, error) {
	return wrappedAtoi(getStringImpl(name, id3v2.UserFrames))
}

func (id3v2 *ID3v22) SetTitle(title string) error {
	return id3v2.SetString("TT2", title)
}

func (id3v2 *ID3v22) SetArtist(artist string) error {
	return id3v2.SetString("TP1", artist)
}

func (id3v2 *ID3v22) SetAlbum(album string) error {
	return id3v2.SetString("TAL", album)
}

func (id3v2 *ID3v22) SetYear(year int) error {
	setStringImpl("TYE", fmt.Sprintf("%d", year), id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) SetComment(comment string) error {
	 id3v2.SetString("COM", comment)
	return nil

}

// TODO(rjk): It's arguable that this is wrong. 
func (id3v2 *ID3v22) SetGenre(genre string) error {
	gen, err := GetGenreByName(genre)
	if err != nil {
		return err
	}

	// It's unclear to me if this will roundtrip correctly. I think so?
	return id3v2.SetString("TCO", fmt.Sprintf("(%d)", gen))
}

func (id3v2 *ID3v22) SetAlbumArtist(albumArtist string) error {
	return fmt.Errorf("SetAlbumArtist not available in ID3v22")
}

func (id3v2 *ID3v22) SetDate(date time.Time) error {
	panic("implement me")
}

func (id3v2 *ID3v22) SetArranger(arranger string) error {
	 setStringImpl("TP4", arranger, id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) SetAuthor(author string) error {
	 setStringImpl("TOL",author, id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) SetBPM(bmp int) error {
	 setStringImpl("TBP",fmt.Sprintf("%d", bmp), id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) SetCatalogNumber(catalogNumber string) error {
	 setStringImpl("CATALOGNUMBER",catalogNumber, id3v2.UserFrames)
	return nil
}

func (id3v2 *ID3v22) SetCompilation(compilation string) error {
	return  fmt.Errorf("SetCompilation not available in ID3v22")
}

func (id3v2 *ID3v22) SetComposer(composer string) error {
	 setStringImpl("TCM",composer, id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) SetConductor(conductor string) error {
	 setStringImpl("TP3",conductor, id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) SetCopyright(copyright string) error {
	 setStringImpl("TCR",copyright, id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) SetDescription(description string) error {
	 setStringImpl("TT3",description, id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) SetDiscNumber(number int, total int) error {
	 setStringImpl("TPA",fmt.Sprintf("%d/%d", number, total), id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) SetEncodedBy(encodedBy string) error {
	return id3v2.SetString("TEN", encodedBy)
}

func (id3v2 *ID3v22) SetTrackNumber(number int, total int) error {
	 setStringImpl("TRK",fmt.Sprintf("%d/%d", number, total), id3v2.Frames)
	return nil
}

// TODO(rjk): Refactor to share implementation with id3v24.
func (id3v2 *ID3v22) SetPicture(picture image.Image) error {
	// Only PNG
	buf := new(bytes.Buffer)
	err := png.Encode(buf, picture)
	if err != nil {
		return err
	}

	attacheched, err := id3v2.GetAttachedPicture()
	if err != nil {
		// Set default params
		newPicture := &AttachedPicture{
			MIME:        "image/png",
			PictureType: 2, // Other file info
			Description: "",
			Data:        buf.Bytes(),
		}
		setAttachedPictureImpl("PIC", newPicture, id3v2.Frames)
		return nil
	}
	// save metainfo
	attacheched.MIME = mimeImagePNG
	attacheched.Data = buf.Bytes()

	setAttachedPictureImpl("PIC", attacheched, id3v2.Frames)
	return nil
}

func (id3v2 *ID3v22) DeleteTag(name string) error {
	delete(id3v2.Frames, name)
	return nil
}

func (id3v2 *ID3v22) DeleteAll() error {
	id3v2.Frames = make(map[string][]byte)
	return nil
}

func (id3v2 *ID3v22) DeleteTitle() error {
	return id3v2.DeleteTag("TT2")
}

func (id3v2 *ID3v22) DeleteArtist() error {
	return id3v2.DeleteTag("TP1")
}

func (id3v2 *ID3v22) DeleteAlbum() error {
	return id3v2.DeleteTag("TAL")
}

func (id3v2 *ID3v22) DeleteYear() error {
	return id3v2.DeleteTag("TYE")
}

func (id3v2 *ID3v22) DeleteComment() error {
	// TODO(rjk): NB the need to sort this out.
	return id3v2.DeleteTag("COM")
}

func (id3v2 *ID3v22) DeleteGenre() error {
	return id3v2.DeleteTag("TCO")
}

func (id3v2 *ID3v22) DeleteAlbumArtist() error {
	return  fmt.Errorf("DeleteAlbumArtist not available in ID3v22")
}

func (id3v2 *ID3v22) DeleteDate() error {
	panic("implement me")
}

func (id3v2 *ID3v22) DeleteArranger() error {
	return id3v2.DeleteTag("TP4")
}

func (id3v2 *ID3v22) DeleteAuthor() error {
	return id3v2.DeleteTag("TOL")
}

func (id3v2 *ID3v22) DeleteBPM() error {
	return id3v2.DeleteTag("TBP")
}

func (id3v2 *ID3v22) DeleteCatalogNumber() error {
	delete(id3v2.UserFrames, "CATALOGNUMBER")
	return nil
}

func (id3v2 *ID3v22) DeleteCompilation() error {
	return  fmt.Errorf("DeleteCompilation not available in ID3v22")
}

func (id3v2 *ID3v22) DeleteComposer() error {
	return id3v2.DeleteTag("TCM")
}

func (id3v2 *ID3v22) DeleteConductor() error {
	return id3v2.DeleteTag("TP3")
}

func (id3v2 *ID3v22) DeleteCopyright() error {
	return id3v2.DeleteTag("TCR")
}

func (id3v2 *ID3v22) DeleteDescription() error {
	return id3v2.DeleteTag("TT3")
}

func (id3v2 *ID3v22) DeleteDiscNumber() error {
	return id3v2.DeleteTag("TPA")
}

func (id3v2 *ID3v22) DeleteEncodedBy() error {
	return id3v2.DeleteTag("TEN")
}

func (id3v2 *ID3v22) DeleteTrackNumber() error {
	return id3v2.DeleteTag("TRK")
}

func (id3v2 *ID3v22) DeletePicture() error {
	return id3v2.DeleteTag("PIC")
}

// TODO(rjk): Refactor me. 
func (id3v2 *ID3v22) SaveFile(path string) error {
	return saveFileImpl(id3v2, path)
}

func sizePackID3v22(length int) []byte {
	header := make([]byte, id3v22FrameHeaderSize)
		header[0] = byte(length >> 16)
		header[1] = byte(length >> 8)
		header[2] = byte(length)
	return header
}

func (id3v2 *ID3v22) Save(writer io.WriteSeeker) error {
	id3v22packings := AllHeaderFields{
		 []byte(id3MarkerValue),
		 []byte{2, 0, 0},
		IntToByteSynchsafe(getFramesLength(id3v2.Frames) + getFramesLength(id3v2.UserFrames)),
	}

	// write header
	err := writeHeaderImpl(writer, id3v22packings)
	if err != nil {
		return err
	}

	// write tags
	err =  writeFramesImpl(writer, id3v2.Frames, sizePackID3v22)
	if err != nil {
		return err
	}
	err =  writeFramesImpl(writer,  id3v2.UserFrames, sizePackID3v22)
	if err != nil {
		return err
	}

	// write data
	_, err = writer.Write(id3v2.Data)
	if err != nil {
		return err
	}
	return nil
}

// nolint:gocyclo
func ReadID3v22(input io.ReadSeeker) (*ID3v22, error) {
	header := ID3v22{}

	if input == nil {
		return nil, ErrEmptyFile
	}

	// Seek to file start
	startIndex, err := input.Seek(0, io.SeekStart)
	if startIndex != 0 {
		return nil, ErrSeekFile
	}

	if err != nil {
		return nil, err
	}

	// Header size
	headerByte := make([]byte, 10)
	nReaded, err := input.Read(headerByte)
	if err != nil {
		return nil, err
	}
	if nReaded != 10 {
		return nil, errors.New("error header length")
	}

	// Marker
	marker := string(headerByte[0:3])
	if marker != id3MarkerValue {
		return nil, errors.New("error file marker")
	}
	header.Marker = marker

	// Version
	versionByte := headerByte[3]
	if versionByte != 2 {
		return nil, ErrUnsupportedFormat
	}
	// Sub version is 0.

	// Flags
	// TODO(rjk): Read flags here.
	// header.Flags = id3v23Flags(headerByte[5])

	// Length
	length := ByteToIntSynchsafe(headerByte[6:10])
	header.Length = length

	// Extended headers
	header.Frames = make(map[string][]byte)
	curRead := 0
	for curRead < length {
		bytesExtendedHeader := make([]byte, id3v22FrameHeaderSize)
		nReaded, err = input.Read(bytesExtendedHeader)
		if err != nil {
			return nil, err
		}
		if nReaded != id3v22FrameHeaderSize {
			return nil, errors.New("error extended header length")
		}
		// Frame identifier
		key := string(bytesExtendedHeader[0:3])

		// Frame data size
		size := ByteToInt(bytesExtendedHeader[3:id3v22FrameHeaderSize])

		bytesExtendedValue := make([]byte, size)
		nReaded, err = input.Read(bytesExtendedValue)
		if err != nil {
			return nil, err
		}
		if nReaded != size {
			return nil, errors.New("error extended value length")
		}

		// This block is intended to truncate the text frame.
		// TODO(rjk): Align with the v23 implementation.
		if key[0:1] == "T" {
			pos := -1
			for i, v := range bytesExtendedValue {
				if v == 0 && i > 0 {
					pos = i
				}
			}
			if pos != -1 {
				bytesExtendedValue = bytesExtendedValue[0:pos]
			}
		}

		header.Frames[key] = bytesExtendedValue
		// TODO(rjk): handle user frames!

		/*
			header.Frames = append(header.Frames, ID3v22Frame{
				key,
				bytesExtendedValue,
			})
		*/

		curRead += id3v22FrameHeaderSize + size
	}
	return &header, nil
}

func checkID3v22(input io.ReadSeeker) bool {
	if input == nil {
		return false
	}

	// read marker (3 bytes) and version (1 byte) for ID3v2
	data, err := seekAndRead(input, 0, io.SeekStart, 4)
	if err != nil {
		return false
	}
	marker := string(data[0:3])

	// id3v2
	if marker != id3MarkerValue {
		return false
	}

	versionByte := data[3]
	return versionByte == 2
}

func (id3v2 *ID3v22) GetString(name string) (string, error) {
	return getStringImpl(name, id3v2.Frames)
}

func (id3v2 *ID3v22) GetBytes(name string) ([]byte, error) {
	return getBytesImpl(name, id3v2.Frames)
}

func (id3v2 *ID3v22) SetString(name string, value string) error {
	setStringImpl(name, value, id3v2.Frames)
	return nil
}
