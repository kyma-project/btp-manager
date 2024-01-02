#!/bin/bash

count_success=0
count_fail=0

while true; do
    # Execute the make test command
    make test

    # Check the exit status of the last command
    if [ $? -eq 0 ]; then
        ((count_success++))
        echo "Test succeeded"
    else
        ((count_fail++))
        echo "Test failed"
    fi

    # Print the current variables
    echo "Number of successful tests: $count_success"
    echo "Number of failed tests: $count_fail"

    # Wait for 1 second before next iteration
    sleep 1
done