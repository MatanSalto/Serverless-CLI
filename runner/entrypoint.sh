#!/bin/sh
set -e

# Run the Python file given by SLP_ENTRYPOINT from /opt/code
if [ -z "$SLP_ENTRYPOINT" ]; then
    echo "SLP_ENTRYPOINT is not set"
    exit 1
fi

SCRIPT="/opt/code/$SLP_ENTRYPOINT"
if [ ! -f "$SCRIPT" ]; then
    echo "SLP_ENTRYPOINT file not found: $SCRIPT" 
    exit 1
fi

exec python -u "$SCRIPT"
