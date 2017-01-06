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
; v6    = score
; v7    = direction (0=up, 1=right, 2=down, 3=left)
; v8    = food x
; v9    = food y
; va    = snake head
; vb    = snake tail
; vc    = snake life
;
;! use WASD to move the snake
;! and eat the pellets
;

    cls

    ; the snake has an initial length of 2
    ld          va, 4
    ld          vb, 0
    ld          v6, 0

    ; load the initial snake tail and head into memory
    ld          i, snake_tail
    ld          v5, [i]

    ; start moving to the right
    ld          v7, 1

    ; draw the initial snake
    ld          i, start
    drw         v0, v1, 1

    ; spawn the initial fool pellet
    call        spawn_food

    ; show the initial score
    call        draw_score

loop:
    call        user_input
    call        move
    call        write_head
    call        check_bounds
    call        draw_head
    call        erase_tail

    jp          loop

user_input:
    ld          v0, 0
    sknp        v0

    break
    scl
    ld          v0, 1
    sknp        v0
    break
    scr


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

    ; erase the current score and increment
    call        draw_score
    add         v6, 1
    call        draw_score

    ; grow the snake by 2.. do this at the tail
    ; and write two dummy positions into memory
    add         vb, #fc ; -4
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
    sne         v4, #ff
    jp          game_over
    sne         v4, 64
    jp          game_over
    sne         v5, #ff
    jp          game_over
    sne         v5, 32
    jp          game_over
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

draw_score:
    ld          i, score
    ld          b, v6
    ld          v2, [i]

    ; where to draw the score
    ld          v0, 55
    ld          v3, 0

    ; tens digit
    ld          f, v1
    drw         v0, v3, 5

    ; ones digit
    ld          f, v2
    add         v0, 5
    drw         v0, v3, 5

    ret

fill_life:
    ld          vc, #1f

    ; draw the life bar to full at top
    ld          i, life_bar
    ld          v0, 0
    ld          v1, 0
rep:
    drw         v0, v1, 1
    add         v0, 8
    se          v0, #40
    jp          rep
    ret

game_over:
    ld          v0, 30
    ld          st, v0

done:
    jp          done


life_bar:
    dw          #F0F0,#F0F0,#F0F0,#F0F0,#F0F0,#F0F0,#F0F0,#F0F0
start:
    db          $111.....
dot:
    db          $1.......

score:
    db          0, 0, 0

snake_tail:
    db          10, 10, 11, 10, 12, 10
