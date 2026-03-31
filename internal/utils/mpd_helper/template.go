package mdp_helper

import (
	"encoding/xml"
)

// MPD represents the root element of the DASH Media Presentation Description
type MPD struct {
	XMLName                   xml.Name           `xml:"MPD"`
	XMLNS                     string             `xml:"xmlns,attr"`
	XMLNSXSI                  string             `xml:"xmlns:xsi,attr"`
	XMLNSXlink                string             `xml:"xmlns:xlink,attr"`
	XSISchemaLocation         string             `xml:"xsi:schemaLocation,attr"`
	Profiles                  string             `xml:"profiles,attr"`
	Type                      string             `xml:"type,attr"`
	MediaPresentationDuration string             `xml:"mediaPresentationDuration,attr"`
	MaxSegmentDuration        string             `xml:"maxSegmentDuration,attr"`
	MinBufferTime             string             `xml:"minBufferTime,attr"`
	ProgramInformation        ProgramInformation `xml:"ProgramInformation"`
	ServiceDescription        ServiceDescription `xml:"ServiceDescription"`
	Periods                   []*Period          `xml:"Period"`
}

// ProgramInformation contains descriptive information about the program
type ProgramInformation struct {
	Title string `xml:"Title"`
}

// ServiceDescription contains service description information
type ServiceDescription struct {
	ID string `xml:"id,attr"`
}

// Period represents a media content period
type Period struct {
	ID             string           `xml:"id,attr"`
	Start          string           `xml:"start,attr"`
	AdaptationSets []*AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents a set of interchangeable encoded versions of media content
type AdaptationSet struct {
	ID                 string            `xml:"id,attr"`
	ContentType        string            `xml:"contentType,attr"`
	StartWithSAP       int               `xml:"startWithSAP,attr"`
	SegmentAlignment   bool              `xml:"segmentAlignment,attr"`
	BitstreamSwitching bool              `xml:"bitstreamSwitching,attr"`
	FrameRate          string            `xml:"frameRate,attr,omitempty"`
	MaxWidth           int32             `xml:"maxWidth,attr,omitempty"`
	MaxHeight          int32             `xml:"maxHeight,attr,omitempty"`
	PAR                string            `xml:"par,attr,omitempty"`
	Lang               string            `xml:"lang,attr,omitempty"`
	Representations    []*Representation `xml:"Representation"`
}

// Representation represents a single media stream
type Representation struct {
	ID                        string                     `xml:"id,attr"`
	MimeType                  string                     `xml:"mimeType,attr"`
	Codecs                    string                     `xml:"codecs,attr"`
	Bandwidth                 int32                      `xml:"bandwidth,attr"`
	Width                     int32                      `xml:"width,attr,omitempty"`
	Height                    int32                      `xml:"height,attr,omitempty"`
	SAR                       string                     `xml:"sar,attr,omitempty"`
	AudioSamplingRate         string                     `xml:"audioSamplingRate,attr,omitempty"`
	AudioChannelConfiguration *AudioChannelConfiguration `xml:"AudioChannelConfiguration,omitempty"`
	SegmentList               *SegmentList               `xml:"SegmentList"`
	BaseUrl                   string                     `xml:"BaseURL,omitempty"`
}

// AudioChannelConfiguration provides information about the audio channel configuration
type AudioChannelConfiguration struct {
	SchemeIDURI string `xml:"schemeIdUri,attr"`
	Value       string `xml:"value,attr"`
}

// SegmentList contains the list of segment URLs for a representation
type SegmentList struct {
	Timescale      int             `xml:"timescale,attr"`
	Duration       int             `xml:"duration,attr"`
	StartNumber    int             `xml:"startNumber,attr"`
	Initialization *Initialization `xml:"Initialization"`
	SegmentURLs    []*SegmentURL   `xml:"SegmentURL"`
}

// Initialization provides information about the initialization segment
type Initialization struct {
	SourceURL string `xml:"sourceURL,attr"`
}

// SegmentURL represents a media segment URL
type SegmentURL struct {
	Media string `xml:"media,attr"`
}
