#!/usr/bin/env bash

rm test.yaml

CHANNEL=fast TAG=0.0 ./update_channel_mapping.sh test.yaml >/dev/null

if ! diff test.yaml testdata/channel.0.yaml >/dev/null; then
 { echo "Channel mapping for fast channel with tag 0.0 does not match expected output"; exit 1; }
fi

CHANNEL=fast TAG=1.0 ./update_channel_mapping.sh test.yaml >/dev/null

if ! diff test.yaml testdata/channel.1.yaml >/dev/null; then
 { echo "Channel mapping for fast channel with tag 1.0 does not match expected output"; exit 1; }
fi

CHANNEL=regular TAG=1.0 ./update_channel_mapping.sh test.yaml >/dev/null

if ! diff test.yaml testdata/channel.2.yaml >/dev/null; then
 { echo "Channel mapping for regular channel with tag 1.0 does not match expected output"; exit 1; }
fi

if ./update_channel_mapping.sh test.yaml >/dev/null; then
  echo "Channel mapping update without parameters succeeded, but it should not have"
  exit 1
fi

echo "Channel mapping test passed successfully"
