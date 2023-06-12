#!/bin/bash
cd "$(dirname "$0")"

awk '/gophers_reasons_section_start/,/gophers_reasons_section_end/' < ../../controllers/conditions.go | grep '='
printf "===="

awk '/gophers_metadata_section_start/,/gophers_metadata_section_end/' < ../../controllers/conditions.go | grep ':'
printf "===="

awk '/table_start/,/table_end/' < ../../docs/contributor/02-10-operations.md | grep "|"