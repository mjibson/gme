/*
Package gme decodes game music files.

This package requires cgo and uses code from
http://blargg.8bitalley.com/libs/audio.html#Game_Music_Emu.
*/
package gme

/*
#include "gme.h"
static short short_index(short *s, int i) {
  return s[i];
}
*/
import "C"

import (
	"fmt"
	"io"
	"time"
	"unsafe"
)

// New opens the file from b with given sample rate.
func New(b []byte, sampleRate int) (*GME, error) {
	var g GME
	data := unsafe.Pointer(&b[0])
	cerror := C.gme_open_data(data, C.long(len(b)), &g.emu, C.long(sampleRate))
	if err := gmeError(cerror); err != nil {
		return nil, err
	}
	return &g, nil
}

// GME decodes game music.
type GME struct {
	emu *_Ctype_struct_Music_Emu
}

type Track struct {
	// Times; negative if unknown.
	Length      time.Duration
	IntroLength time.Duration
	LoopLength  time.Duration

	System    string
	Game      string
	Song      string
	Author    string
	Copyright string
	Comment   string
	Dumper    string
}

// Tracks returns the number of tracks in the file.
func (g *GME) Tracks() int {
	return int(C.gme_track_count(g.emu))
}

// Track returns information about the n-th track, 0-based.
func (g *GME) Track(track int) (Track, error) {
	var t _Ctype_struct_track_info_t
	cerror := C.gme_track_info(g.emu, &t, C.int(track))
	if err := gmeError(cerror); err != nil {
		return Track{}, err
	}
	return Track{
		Length:      time.Duration(t.length) * time.Millisecond,
		IntroLength: time.Duration(t.intro_length) * time.Millisecond,
		LoopLength:  time.Duration(t.loop_length) * time.Millisecond,
		System:      cstring(t.system),
		Game:        cstring(t.game),
		Song:        cstring(t.song),
		Author:      cstring(t.game),
		Copyright:   cstring(t.copyright),
		Comment:     cstring(t.comment),
		Dumper:      cstring(t.dumper),
	}, nil
}

func cstring(s [256]C.char) string {
	str := (*C.char)(unsafe.Pointer(&s[0]))
	return C.GoString(str)
}

// Start initializes the n-th track for playback, 0-based.
func (g *GME) Start(track int) error {
	C.gme_ignore_silence(g.emu, C.int(0))
	return gmeError(C.gme_start_track(g.emu, C.int(track)))
}

// Played returns the played time of the current track.
func (g *GME) Played() time.Duration {
	return time.Duration(C.gme_tell(g.emu)) * time.Millisecond
}

// Ended returns whether the current track has ended.
func (g *GME) Ended() bool {
	return C.gme_track_ended(g.emu) == 1
}

// Play decodes the next samples into data. Data is populated with two channels
// interleaved.
func (g *GME) Play(data []int16) (err error) {
	b := make([]C.short, len(data))
	datablock := (*C.short)(unsafe.Pointer(&b[0]))
	cerror := C.gme_play(g.emu, C.long(len(b)), datablock)
	if err := gmeError(cerror); err != nil {
		return err
	}
	for i := range data {
		data[i] = int16(C.short_index(datablock, C.int(i)))
	}
	if false && g.Ended() {
		return io.EOF
	}
	return nil
}

// Close closes the GME file and frees its used memory.
func (g *GME) Close() {
	if g.emu != nil {
		C.gme_delete(g.emu)
		g.emu = nil
	}
}

// Warning returns the last warning produced and clears it.
func (g *GME) Warning() string {
	if g == nil || g.emu == nil {
		return ""
	}
	return C.GoString(C.gme_warning(g.emu))
}

func gmeError(e _Ctype_gme_err_t) error {
	if e == nil {
		return nil
	}
	return fmt.Errorf("gme: %v", C.GoString(e))
}