; tests all the permutations of LD instructions
;

empty:  jp empty

        ld v0, #ff      ; load immediate
        ld v1, v2       ; load register

        ld i, #400      ; load address with immediate
        ld i, empty     ; load address with label

        ld v0, d        ; read delay timer
        ld v1, dt

        ld d, v1        ; load delay timer
        ld dt, v2

        ld s, v1        ; load sound timer
        ld st, v2

        ld f, v5        ; load font sprite
        ld b, v6        ; load bcd

        ld [i], vf      ; save registers
        ld vf, [i]      ; load registers
