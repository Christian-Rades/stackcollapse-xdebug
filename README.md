# stackcollapse-xdebug
A tool to collapse xdebug function traces to generate flamegraphs.

Usage:

`./stackcollapse-php < trace.xt > trace.collapsed`

Then the collapsed stack can be used to generate a flamegraph with either https://github.com/brendangregg/FlameGraph or with https://github.com/jonhoo/inferno
