#!/usr/bin/env bash

        OK=0
        FAIL=0
        while true; do
          make -i test > /dev/null
          if [ $? -eq 0 ]; then
            ((OK++))
            echo "(COUNT OF) OK -> $OK"
          else
            ((FAIL++))
            echo "(COUNT OF) FAIL -> $FAIL"
          fi
        done