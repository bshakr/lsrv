#!/bin/bash

echo "Running lsrv benchmark (5 iterations)..."
echo ""

for i in 1 2 3 4 5; do
    echo -n "Run $i: "
    /usr/bin/time -p ~/go/bin/lsrv > /dev/null
done

echo ""
echo "Benchmark complete!"
