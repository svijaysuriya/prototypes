#!/bin/bash

HOST="localhost"
PORT="8080"
COUNT=5

echo "Starting $COUNT nc clients connecting to $HOST on port $PORT..."

for i in $(seq 1 $COUNT); do
  echo "Starting client $i..."
  # Run the client in the background. Input can be piped or handled via redirection.
  # For interactive use, this will be non-ideal.
  { echo "event loop $i"; sleep 1; } | nc "$HOST" "$PORT" &
#   sleep 1
done

echo "All clients started. Check 'jobs' for status."
