#!/usr/bin/env bats

setup_file() {
    PROJECT_ROOT="$(cd "${BATS_TEST_DIRNAME}/.." && pwd)"
    export PROJECT_ROOT
}

setup() {
    export MOLE_BASE_LOADED=1 # mock base loaded
    # Mocking base functions used in ui.sh if any (mostly spinner/logging stuff, unlikely to affect read_key)
}

@test "read_key maps j/k/h/l to navigation" {
    # Source the UI library
    source "$PROJECT_ROOT/lib/core/ui.sh"

    # Test j -> DOWN
    run bash -c "source '$PROJECT_ROOT/lib/core/ui.sh'; echo -n 'j' | read_key"
    [ "$output" = "DOWN" ]

    # Test k -> UP
    run bash -c "source '$PROJECT_ROOT/lib/core/ui.sh'; echo -n 'k' | read_key"
    [ "$output" = "UP" ]

    # Test h -> LEFT
    run bash -c "source '$PROJECT_ROOT/lib/core/ui.sh'; echo -n 'h' | read_key"
    [ "$output" = "LEFT" ]

    # Test l -> RIGHT
    run bash -c "source '$PROJECT_ROOT/lib/core/ui.sh'; echo -n 'l' | read_key"
    [ "$output" = "RIGHT" ]
}

@test "read_key maps uppercase J/K/H/L to navigation" {
    run bash -c "source '$PROJECT_ROOT/lib/core/ui.sh'; echo -n 'J' | read_key"
    [ "$output" = "DOWN" ]

    run bash -c "source '$PROJECT_ROOT/lib/core/ui.sh'; echo -n 'K' | read_key"
    [ "$output" = "UP" ]
}

@test "read_key respects MOLE_READ_KEY_FORCE_CHAR" {
    # When force char is on, j should return CHAR:j
    run bash -c "export MOLE_READ_KEY_FORCE_CHAR=1; source '$PROJECT_ROOT/lib/core/ui.sh'; echo -n 'j' | read_key"
    [ "$output" = "CHAR:j" ]
}
