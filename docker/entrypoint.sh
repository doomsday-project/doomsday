#!/bin/bash
config=/doomsday/ddayconfig-minimal.yml

if [ -f "$DDAY_CONFIG_FILE" ]; then
  echo "Using specified config file: $config"
  config=$DDAY_CONFIG_FILE
fi

if [ ! -z "$DDAY_CONFIG" ]; then
  echo "Using custom config from environment..."
  echo "$DDAY_CONFIG" > "$config"
fi

/doomsday/doomsday server -m $config