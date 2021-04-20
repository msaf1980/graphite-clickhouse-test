# graphite-clickhouse-test
Some wrapper for run carbon-clickhouse/graphite-clickhouse tests

Quick and dirty integration tests for carbon-clickhouse/graphite-clickhouse

Usage:
1) Install docker (for clickhouse)
2) Build carbon-clickhouse and graphite-clickhouse
3) Run tests (clickhouse configuration and rules examples in tests)
$ ./graphite-clickhouse-test -config tests/rollup
