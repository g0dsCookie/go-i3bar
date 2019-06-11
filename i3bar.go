package i3bar

import (
	"encoding/json"
	"io"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// Header structure used to initialize the i3bar protocol stream.
type Header struct {
	// Version to use for your protocol stream.
	Version int `json:"version"`

	// StopSignal i3bar should send to stop our processing.
	// Defaults to syscall.SIGSTOP
	StopSignal int `json:"stop_signal,omitempty"`

	// ContSignal i3bar should send to continue our processing.
	// Defaults to syscall.SIGCONT
	ContSignal int `json:"cont_signal,omitempty"`

	// ClickEvents defines if i3bar should send an infinite array
	// to stdin with click events.
	ClickEvents bool `json:"click_events,omitempty"`
}

// Alignment within a Block.
type Alignment int

const (
	// Left align the block.
	Left Alignment = iota
	// Center align the block.
	Center
	// Right align the block.
	Right
)

// UnmarshalText decodes a human-readable string value
// into it's computational Alignment value.
func (a *Alignment) UnmarshalText(b []byte) error {
	switch strings.ToLower(string(b)) {
	case "left":
		*a = Left
	case "center":
		*a = Center
	case "right":
		*a = Right
	default:
		return errors.Errorf("unknown alignment: %s", string(b))
	}
	return nil
}

// MarshalText encodes the computational Alignment value
// into human and i3bar readable string value.
func (a Alignment) MarshalText() ([]byte, error) {
	var align string
	switch a {
	case Left:
		align = "left"
	case Center:
		align = "center"
	case Right:
		align = "right"
	default:
		return nil, errors.Errorf("unknown alignment: %d", a)
	}
	return []byte(align), nil
}

// Markup to specify how a block should be parsed.
type Markup int

const (
	// NoMarkup specifies to not use any parser.
	NoMarkup Markup = iota
	// Pango specifies to use the Pango markup language.
	// See also https://developer.gnome.org/pango/stable/PangoMarkupFormat.html
	Pango
)

// UnmarshalText decodes a human-readable string value
// into it's computational Markup value.
func (m *Markup) UnmarshalText(b []byte) error {
	switch strings.ToLower(string(b)) {
	case "none":
		*m = NoMarkup
	case "pango":
		*m = Pango
	default:
		return errors.Errorf("unknown markup: %s", string(b))
	}
	return nil
}

// MarshalText encodes the computational Markup value
// into human and i3bar readable string value.
func (m Markup) MarshalText() ([]byte, error) {
	var markup string
	switch m {
	case NoMarkup:
		markup = "none"
	case Pango:
		markup = "pango"
	default:
		return nil, errors.Errorf("unknown markup: %d", m)
	}
	return []byte(markup), nil
}

// Block specifies a single block within a StatusLine.
type Block struct {
	// Name to identify this block.
	Name string `json:"name,omitempty"`

	// Instance of this block.
	Instance string `json:"instance,omitempty"`

	// FullText to display in this block.
	FullText string `json:"full_text"`

	// ShortText to display if the status line needs to be shortened.
	ShortText string `json:"short_text,omitempty"`

	// Color of the text in hex. (#rrggbb)
	Color string `json:"color,omitempty"`

	// Background color in hex. (#rrggbb)
	Background string `json:"background,omitempty"`

	// Border color in hex. (#rrggbb)
	Border string `json:"border,omitempty"`

	// MinWidth specifies the minimum width of the block in pixels.
	// You can also specify a text representing the longest possible text.
	MinWidth string `json:"min_width,omitempty"`

	// Align text on the center, right or left.
	Align Alignment `json:"align,omitempty"`

	// Urgent specifies if the current value is urgent.
	Urgent bool `json:"urgent,omitempty"`

	// Separator specifies if a separator line should be drawn after this block.
	Separator bool `json:"separator,omitempty"`

	// SeparatorBlockWidth specified the amount of pixels to leave black after the block.
	SeparatorBlockWidth int `json:"separator_block_width,omitempty"`

	// Markup specifies how the block should be parsed.
	Markup Markup `json:"markup,omitempty"`
}

// StatusLine represents a full i3bar status line.
type StatusLine []*Block

// Stream represents an i3bar protocol stream.
type Stream struct {
	w    io.Writer
	e    *json.Encoder
	wMux sync.Mutex

	r io.Reader
	d *json.Decoder
}

// NewStream initializes a new i3bar protocol stream with specified parameters.
// w is the io.Writer where to send the infinite Block json array.
// r is the io.Reader where to read the infinite ClickEvent json array. (TBD)
// pretty can be true if you want the json encoder to pretty-print the json.
// h is the Header which is used to initialize the i3bar protocol.
func NewStream(w io.Writer, r io.Reader, pretty bool, h Header) (*Stream, error) {
	stream := &Stream{
		w:    w,
		e:    json.NewEncoder(w),
		wMux: sync.Mutex{},
		r:    r,
		d:    json.NewDecoder(r),
	}

	if pretty {
		stream.e.SetIndent("", "    ")
	}

	// send protocol header
	if err := stream.e.Encode(h); err != nil {
		return nil, errors.Wrap(err, "Failed to send header")
	}

	// start infinite loop on writer
	if _, err := w.Write([]byte("[")); err != nil {
		return nil, errors.Wrap(err, "Failed to start infinite json array")
	}

	// TODO: start reader

	return stream, nil
}

// SendLine sends a new status line to the underlying stream.
// This function is thread safe.
func (s *Stream) SendLine(b StatusLine) error {
	s.wMux.Lock()
	defer s.wMux.Unlock()
	if err := s.e.Encode(b); err != nil {
		return errors.Wrap(err, "Failed to encode status line into json stream")
	}
	return nil
}

// Close closes the underlying stream by issuing an ]
// to close the infinite json array.
//
// This function is thread safe although you don't want to call
// this multiple times. Also you don't want to call any other method
// after this.
func (s *Stream) Close() error {
	s.wMux.Lock()
	defer s.wMux.Unlock()
	if _, err := s.w.Write([]byte("]")); err != nil {
		return errors.Wrap(err, "Failed to close infinite json array")
	}
	return nil
}
