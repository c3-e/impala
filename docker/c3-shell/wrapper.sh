#!/bin/bash

# Start the first process
first=goterm
cd /opt/impala/bin/goterm && ./goterm -authcallback ${AUTH_CALLBACK} &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start $first process: $status"
  exit $status
fi

# Start the second process
second=golog
cd /opt/impala/bin/golog && ./golog -listen=0.0.0.0:3010 -impalad=impalad &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start $second process: $status"
  exit $status
fi

cd /opt/impala/data

while sleep 60; do
  ps aux | grep $first | grep -q -v grep
  PROCESS_1_STATUS=$?
  ps aux | grep $second | grep -q -v grep
  PROCESS_2_STATUS=$?
  # If the greps above find anything, they exit with 0 status
  # If they are not both 0, then something is wrong
  if [ $PROCESS_1_STATUS -ne 0 -o $PROCESS_2_STATUS -ne 0 ]; then
    echo "One of the processes has already exited."
    exit 1
  fi
done
