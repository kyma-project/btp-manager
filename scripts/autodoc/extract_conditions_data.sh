#!/bin/bash
cd "$(dirname "$0")"

FILE_CONDITIONS=../../internal/conditions/conditions.go
if [ -f "$FILE_CONDITIONS" ]; then
    awk '/gophers_reasons_section_start/,/gophers_reasons_section_end/' < $FILE_CONDITIONS | grep '='
    printf "===="

    awk '/gophers_metadata_section_start/,/gophers_metadata_section_end/' < $FILE_CONDITIONS | grep ':'
    printf "===="
else
    >&2 echo "Error because $FILE_CONDITIONS file does not exist. "
    exit 1
fi

FILE_OPERATIONS=../../docs/contributor/02-10-operations.md
if [ -f "$FILE_OPERATIONS" ]; then
    awk '/table_start/,/table_end/' < $FILE_OPERATIONS | grep "|"
else
    >&2 echo "Error because $FILE_OPERATIONS file does not exist. "
    exit 1
fi