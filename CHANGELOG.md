# Changes

Want to know what's different between versions? Look no further...

## Version 1.2

__Fixes__

* Window creation flags error with latest SDL. Thanks to @renatorabelo for finding it!

## Version 1.1

___Fixes___

* Fixed errors in README.
* Fixed scanning of string tokens.

___Additions___

* Added CHIP-8E instruction set.
* Added `SUPER` directive, required to use CHIP-48 instructions.
* Added `EXTENDED` directive, required to use CHIP-8E instructions.
* Added `ASCII` directive, for use with CHIP-8E `LD A,VX` instruction.
* Added use of back-quote (`) for text strings.
* Added Step Out functionality (`SHIFT`+`F7`) to debugger.

___Breaking Changes___

* Use of the `SUPER` directive is now required before using CHIP-48 instructions.
* Changed `LD B,VX` instruction to `BCD VX` to match CHIP-8E `BCD VX,VY`.

## Version 1.0

Initial release.
