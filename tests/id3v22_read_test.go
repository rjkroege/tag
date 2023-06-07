package tests

import (
	"github.com/frolovo22/tag"
	"github.com/stretchr/testify/assert"
	"image/png"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"io"
	"fmt"
	"path/filepath"
)

func TestId3v22Read(t *testing.T) {
	asrt := assert.New(t)
	id3, err := tag.ReadFile("id3v2.2.mp3")
	asrt.NoError(err, "open")
	if err != nil {
		return
	}

	asrt.Equal("*tag.ID3v22", reflect.TypeOf(id3).String())

	title, err := id3.GetTitle()
	asrt.NoError(err)
	asrt.Equal("You Are The One", title)

	album, err := id3.GetAlbum()
	asrt.NoError(err)
	asrt.Equal("We Are Pilots", album)

	artist, err := id3.GetArtist()
	asrt.NoError(err)
	asrt.Equal("Shiny Toy Guns", artist)

	year, err := id3.GetYear()
	asrt.NoError(err)
	asrt.Equal(2006, year)

	genre, err := id3.GetGenre()
	asrt.NoError(err)
	asrt.Equal("Alternative", genre)

	encodedBy, err := id3.GetEncodedBy()
	asrt.NoError(err)
	asrt.Equal("iTunes v7.0.2.16", encodedBy)

	picture, err := id3.GetPicture()
	asrt.NoError(err)
	out, err := ioutil.TempFile("", "idv22Tst.jpg")
	asrt.NoError(err)
	defer os.Remove(out.Name())
	err = png.Encode(out, picture)
	asrt.NoError(err)
	cmp := compareFiles("idv22.jpg", out.Name())
	asrt.Equal(true, cmp)

	trackNumber, totalNumber, err := id3.GetTrackNumber()
	asrt.NoError(err)
	asrt.Equal(1, trackNumber)
	asrt.Equal(11, totalNumber)
}

func copyfile(t *testing.T, from, to string) {
	t.Helper()

	sfd, err := os.Open(from)
	if err != nil {
		t.Fatalf("copyfile open %q: %v", from, err)
	}
	dfd, err := os.Create(to)
	if err != nil {
		t.Fatalf("copyfile create %q: %v", to, err)
	}
	
	if _, err := io.Copy(dfd, sfd); err != nil {
		t.Fatalf("copyfile copying: %v", err)
	}
}

func TestId3v22RMW(t *testing.T) {
	testdir := t.TempDir()
	mp3file := "id3v2.2.mp3"

	for _, v := range []struct{
		name string
		mutator  func(s tag.Metadata) error
		checker  func(g tag.Metadata) error
	}{
	{
		name: "Title",
		mutator: func(s tag.Metadata) error {
			return s.SetTitle("binky was here")
		},
		checker: func(s tag.Metadata) error {
			got, err := s.GetTitle()
			if err != nil {
				return nil
			}
			if got != "binky was here" {
				return fmt.Errorf("got: %s, want: %s", got, "binky was here")
			}
			return nil
		},
	},
	} {
		t.Run(v.name, func(t *testing.T) {
			testfile := filepath.Join(testdir, v.name + "-" + mp3file)
			outfile := filepath.Join(testdir, v.name + "-out-" + mp3file)
			copyfile(t, mp3file, testfile)

			// Read the tag data.
			id3, err := tag.ReadFile(testfile)
			if err != nil {
				t.Errorf("can't read tags from %q: %v", testfile, err)
				return
			}
				
			// Modify the tag data.
			err = v.mutator(id3)
			if err != nil {
				t.Errorf("can't mutate %q: %v", testfile, err)
			}

			// Write to a different file.
			err = id3.SaveFile(outfile)
			if err != nil {
				t.Errorf("can't save %q: %v", outfile, err)
			}

// does it even exist
			fi, err := os.Stat(outfile)
			if err != nil {
				t.Errorf("outfile %q doesn't exist", outfile)
			}
			t.Logf("written file %q size %d", outfile, fi.Size())

			// Verify that the update happened
			id3m, err := tag.ReadFile(outfile)
			if err != nil {
				t.Errorf("can't read tags from %q: %v", outfile, err)
				return
			}

			// Check that the mutation happened
			err = v.checker(id3m)
			if err != nil {
				t.Errorf("ch %q: %v", testfile, err)
			}
		})
	}
}
