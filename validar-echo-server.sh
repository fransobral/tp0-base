#!/bin/bash


MESSAGE="Hello Echo"
TIMEOUT=5

RESULT=$(docker run --rm --network tp0_testing_net busybox sh -c "echo '$MESSAGE' | nc -w $TIMEOUT server 12345")

if [ "$RESULT" == "$MESSAGE" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi
