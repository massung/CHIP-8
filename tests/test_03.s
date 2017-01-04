; tests db, dw, and text
;

    ld i, sprite

loop:
    call blit
    jp loop

blit:
    rnd v0, #3f
    rnd v1, #1f
    drw v0, v1, 5
    ret

sprite:
    db $..1111..
    db $.1....1.
    db $.1....1.
    db $.1....1.
    db $..1111..

    dw #ffff
    dw #aaaa

    db "Hello, world!"
