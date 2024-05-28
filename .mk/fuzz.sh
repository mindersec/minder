#!/bin/bash

FUZZ_TIME=${FUZZ_TIME:-30s}
FUZZ_COUNT=${FUZZ_COUNT:-100}
FUZZ_PACKAGE=${FUZZ_PACKAGE:-}
FUZZ_FUNCTION=${FUZZ_FUNCTION:-}

IFS=$'\n'
for line in $(grep -R "func Fuzz" . | grep -v '\.mk'); do
    package=$(echo $line | sed 's%^\(.*\)/[A-Za-z_]*\.go:func .*(.*$%\1%')
    function=$(echo $line | sed 's%^.*/[A-Za-z_]*\.go:func \(.*\)(.*$%\1%')

    if [ -n "$FUZZ_PACKAGE" ] && [ "$package" != "$FUZZ_PACKAGE" ]; then
        continue
    fi
    if [ -n "$FUZZ_FUNCTION" ] && [ "$function" != "$FUZZ_FUNCTION" ]; then
        continue
    fi

    echo "* Fuzzing $package function $function";
    set -x
    go test $package -fuzz -fuzztime=$FUZZ_TIME -count=$FUZZ_COUNT;
    set +x
done