; simple version of snake for chip-8
;
; copyright (c) 2017 by jeffrey massung
; all rights reserved
;
; have fun!
;

; v0-v3 = scratch
; v4    = score
; v5    = x
; v6    = y
; v7    = direction (0=up, 1=right, 2=down, 3=left)
; v8    = food x
; v9    = food y
; v10   = snake length
;
;* use WASD to move the snake
;* and eat the pellets
;

    cls

;;; SETUP
    ld v4, 0
    ld v5, 10
    ld v6, 10
    ld v7, 1


;;; MAIN GAME LOOP
loop:
    call draw_head
    call erase_tail
    call user_input
    call move
    ;call check_bounds

    jp loop


;;; HANDLE INPUT
user_input:
    ld v0, 5 ; w
    sknp v0
    ld v7, 0
    ld v0, 7 ; a
    sknp v0
    ld v7, 3
    ld v0, 8 ; s
    sknp v0
    ld v7, 2
    ld v0, 9 ; d
    sknp v0
    ld v7, 1
    ret


;;; PROCESS MOVEMENT
move:
    ld v0, 1

    ; test against direction
    sne v7, 0
    jp move_up
    sne v7, 1
    jp move_right
    sne v7, 2
    jp move_down
    sne v7, 3
    jp move_left

    ; invalid movement direction
    ret

move_up:
    sub v6, v0
    ret
move_right:
    add v5, v0
    ret
move_down:
    add v6, v0
    ret
move_left:
    sub v5, v0
    ret

;;; DRAW/COLLISION DETECTION
draw_head:
    ld i, dot
    drw v5, v6, 1
    se vf, 1
    ret

    ; the head collided with something
    jp game_over

;;; GET RID OF THE FINAL TAIL PIECE
erase_tail:
    ret

;;; MAKE SURE IN BOUNDS
check_bounds:
    sne v5, #ff
    jp game_over
    sne v5, 64
    jp game_over
    sne v6, #ff
    jp game_over
    sne v6, 32
    jp game_over
    ret

game_over:
    jp game_over


dot:
    db $1.......

score_mem:
    db 0, 0, 0

snake_mem:
    db 0