#!/bin/bash
cd "$(dirname "$0")"
x=$(sh make-module-chart.sh)
echo $x