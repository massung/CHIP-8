; simple version of snake for chip-8
;
; copyright (c) 2017 by jeffrey massung
; all rights reserved
;
; have fun!
;

; v0-v3 = scratch
; v4    = x
; v5    = y
; v7    = direction (0=up, 1=right, 2=down, 3=left)
; v8    = food x
; v9    = food y
; va    = snake head
; vb    = snake tail
;
;! use WASD to move the snake
;! and eat the pellets
;

    cls

    ; the snake has an initial length of 2
    ld          va, 4
    ld          vb, 0

    ;; write the initial snake to memory
    ld          v0, 10
    ld          v1, 10
    ld          v2, 11
    ld          v3, 10
    ld          v4, 12          ; head x
    ld          v5, 10          ; head y
    ld          i, snake_tail
    ld          [i], v5

    ; start moving to the right
    ld          v7, 1

    ; draw the initial snake
    ld          i, start
    drw         v0, v1, 1

    ; spawn the initial fool pellet
    call        spawn_food

loop:
    call        user_input
    call        move
    call        write_head
    call        draw_head
    call        erase_tail
    call        check_bounds

    jp          loop

user_input:
    ld          v0, 5 ; up
    sknp        v0
    ld          v7, 0
    ld          v0, 7 ; left
    sknp        v0
    ld          v7, 3
    ld          v0, 8 ; down
    sknp        v0
    ld          v7, 2
    ld          v0, 9 ; right
    sknp        v0
    ld          v7, 1
    ret

move:
    ld          v0, 1

    ; test against direction
    sne         v7, 0
    jp          move_up
    sne         v7, 1
    jp          move_right
    sne         v7, 2
    jp          move_down
    sne         v7, 3
    jp          move_left

    ; invalid movement direction
    ret

move_up:
    sub         v5, v0
    ret
move_right:
    add         v4, v0
    ret
move_down:
    add         v5, v0
    ret
move_left:
    sub         v4, v0
    ret

write_head:
    ld          i, snake_tail
    add         va, 2
    add         i, va
    ld          v0, v4
    ld          v1, v5
    ld          [i], v1
    ret

draw_head:
    ld          i, dot
    drw         v4, v5, 1
    se          vf, 1
    ret

    ; did the head hit the pellet?
    se          v4, v8
    jp          game_over
    se          v5, v9
    jp          game_over

    ; play a little beep for eating food
    ld          v0, 2
    ld          st, v0

    ; grow the snake by 2.. do this at the tail
    ; and write two dummy positions into memory
    ld          v0, 4
    sub         vb, v0
    ld          i, snake_tail
    add         i, vb
    ld          v0, #ff
    ld          v1, #ff
    ld          v2, #ff
    ld          v3, #ff
    ld          [i], v3

    ; now redraw the head (since it was turned off)
    ld          i, dot
    drw         v4, v5, 1

    ; and spawn more food
    jp          spawn_food

erase_tail:
    ld          i, snake_tail
    add         i, vb
    ld          v1, [i]
    ld          i, dot
    drw         v0, v1, 1
    add         vb, 2
    ret

check_bounds:
    sne v4, #ff
    jp game_over
    sne v4, 64
    jp game_over
    sne v5, #ff
    jp game_over
    sne v5, 32
    jp game_over
    ret

spawn_food:
    rnd         v8, #3f
    rnd         v9, #1f

    ; draw it
    ld          i, dot
    drw         v8, v9, 1

    ; if nothing was there, we're good
    sne         vf, 0
    ret

    ; put it back and try again
    drw         v8, v9, 1
    jp          spawn_food

game_over:
    ; TODO: show some kind of score... ?

done:
    jp done


start:
    db $111.....
dot:
    db $1.......

score:
    db $.111....
    db $1.......
    db $.11.....
    db $...1....
    db $111.....

    db $.111....
    db $1.......
    db $1.......
    db $1.......
    db $.111....

    db $.11.....
    db $1..1....
    db $1..1....
    db $1..1....
    db $.11.....

    db $111.....
    db $1..1....
    db $111.....
    db $1..1....
    db $1..1.....

    db $1111....
    db $1.......
    db $111.....
    db $1.......
    db $1111....

snake_tail:
    db 0

    ; put nothing after this in memory!!