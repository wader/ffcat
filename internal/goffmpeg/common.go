package goffmpeg

import (
	"fmt"
	"reflect"
	"time"

	"github.com/wader/ffcat/internal/goffmpeg/features"
)

// Printer is something that printfs (used for debug logging)
type Printer interface {
	Printf(format string, v ...interface{})
}

// NopPrinter is discard printfer
type NopPrinter struct{}

// Printf nop
func (NopPrinter) Printf(format string, v ...interface{}) {}

// DurationToPosition time.Duration to ffmpeg position format
func DurationToPosition(d time.Duration) string {
	n := uint64(d.Seconds())
	s := n % 60
	n /= 60
	m := n % 60
	n /= 60
	h := n

	return fmt.Sprintf("%d:%.2d:%.2d", h, m, s)
}

// Metadata from libavformat/avformat.h
// json tag is used for metadata key name also
type Metadata struct {
	Album string `json:"album"` // name of the set this work belongs to
	// main creator of the set/album, if different from artist.
	// e.g. "Various Artists" for compilation albums.
	AlbumArtist  string `json:"album_artist"`
	Artist       string `json:"artist"`        // main creator of the work
	Comment      string `json:"comment"`       // any additional description of the file.
	Composer     string `json:"composer"`      // who composed the work, if different from artist.
	Copyright    string `json:"copyright"`     // name of copyright holder.
	CreationTime string `json:"creation_time"` // date when the file was created, preferably in ISO 8601.
	Date         string `json:"date"`          // date when the work was created, preferably in ISO 8601.
	Disc         string `json:"disc"`          // number of a subset, e.g. disc in a multi-disc collection.
	Encoder      string `json:"encoder"`       // name/settings of the software/hardware that produced the file.
	EncodedBy    string `json:"encoded_by"`    // person/group who created the file.
	Filename     string `json:"filename"`      // original name of the file.
	Genre        string `json:"genre"`         // <self-evident>.
	// main language in which the work is performed, preferably
	// in ISO 639-2 format. Multiple languages can be specified by
	// separating them with commas.
	Language string `json:"language"`
	// artist who performed the work, if different from artist.
	// E.g for "Also sprach Zarathustra", artist would be "Richard
	// Strauss" and performer "London Philharmonic Orchestra".
	Performer       string `json:"performer"`
	Publisher       string `json:"publisher"`        // name of the label/publisher.
	ServiceName     string `json:"service_name"`     // name of the service in broadcasting (channel name).
	ServiceProvider string `json:"service_provider"` // name of the service provider in broadcasting.
	Title           string `json:"title"`            // name of the work.
	Track           string `json:"track"`            // number of this work in the set, can be in form current/total.
	VariantBitrate  string `json:"variant_bitrate"`  // the total bitrate of the bitrate variant that the current stream is part of
}

// ToMap convert to a key value string map
func (m Metadata) ToMap() map[string]string {
	kv := map[string]string{}
	t := reflect.TypeOf(m)
	v := reflect.ValueOf(m)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i).String()
		if value == "" {
			continue
		}

		// json tag is used for metadata key name
		key := field.Tag.Get("json")
		kv[key] = value
	}

	return kv
}

// Merge return a new metadata merged with a
func (m Metadata) Merge(a Metadata) Metadata {
	at := reflect.TypeOf(m)
	av := reflect.ValueOf(&m)
	bv := reflect.ValueOf(a)

	for i := 0; i < at.NumField(); i++ {
		af := av.Elem().Field(i)
		if af.String() == "" {
			bf := bv.Field(i)
			af.SetString(bf.String())
		}
	}

	return m
}

// Version return ffmpeg version
func Version() (features.VersionParts, error) {
	return features.Version(FFmpegPath)
}

// Features return description of supported features
func Features() (features.Features, error) {
	return features.LoadFeatures(FFmpegPath)
}
