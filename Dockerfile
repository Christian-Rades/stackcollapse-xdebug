FROM alpine:3

RUN apk add perl git

RUN git clone --depth 1 --branch v1.0 https://github.com/brendangregg/FlameGraph.git 

WORKDIR /FlameGraph


