#!/usr/bin/env bash
set -e

cd "$(dirname "$0")/.."

php -l ref/hmx.php

status=0
for f in test/test_*.php; do
	if php "$f" > /tmp/hmx_test_out 2>&1; then
		echo "PASS $f"
	else
		echo "FAIL $f"
		cat /tmp/hmx_test_out
		status=1
	fi
done

exit $status
