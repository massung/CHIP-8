package main

// typedef unsigned char byte;
// void Tone(void *data, byte *stream, int len);
import "C"
import (
	"reflect"
	"time"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	/// The tone has to ramp in and ramp back out (quickly) or it
	/// will sound weird. This is the current volume of the tone
	/// being generated [0,1]. If the desired volume is > or < than
	/// this value, this value will be adjusted towards the desired
	/// volume and that's what will play.
	///
	Volume float32
)

/// Initialize an audio device for the CHIP-8 virtual machine.
///
func InitAudio() {
	spec := &sdl.AudioSpec {
		Freq: 3000,
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

	// no sound volume
	Volume = 0.0
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

	// ramp the volume to the desired end
	if now < VM.ST {
		Volume = 1.0
	} else {
		if Volume > 0.0 {
			Volume -= 0.25
		}
	}

	// fill in the data with a constant tone
	for i := 0; i < n; i+=4 {
		buf[i] = C.float(Volume)
	}
}
