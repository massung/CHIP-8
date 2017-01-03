package main

// typedef unsigned char byte;
// void Tone(void *data, byte *stream, int len);
import "C"
import (
	"github.com/veandco/go-sdl2/sdl"
	"unsafe"
	"reflect"
	"time"
)

/// Initialize an audio device for the CHIP-8 virtual machine.
///
func InitAudio() {
	spec := &sdl.AudioSpec {
		Freq: 2500,
		Format: sdl.AUDIO_F32,
		Channels: 1,
		Samples: 32,
		Callback: sdl.AudioCallback(C.Tone),
	}

	// open the device and start playing it
	if err := sdl.OpenAudio(spec, nil); err != nil {
		panic(err)
	}

	// start playing the tone immediately
	sdl.PauseAudio(false)
}

//export Tone
func Tone(_ unsafe.Pointer, stream *C.byte, length C.int) {
	p := uintptr(unsafe.Pointer(stream))
	n := int(length)

	// perform the conversion cast
	buf := *(*[]C.float)(unsafe.Pointer(&reflect.SliceHeader{
		Data: p,
		Len: n,
		Cap: n,
	}))

	// get the current time
	now := time.Now().UnixNano()

	// fill in the data with a constant tone
	for i := 0; i < n; i+=4 {
		if now < VM.ST {
			buf[i] = 1.0
		} else {
			buf[i] = 0.0
		}
	}
}
