; test instructions other than loads
;

empty:
    jp empty
    jp v0, empty

    se v0, 10
    se v2, va

    sne v2, $..1111
    sne v3, vf

    skp v0
    sknp v3

    or v0, v1
    and v2, v3
    xor v4,v6

    shr v1
    shl v9

    add v4, #2
    add v5, v7

    sub v1, v3
    subn v4, v5

    rnd v5, 45
    drw v0, v3, 2
