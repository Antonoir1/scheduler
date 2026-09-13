FROM gcr.io/distroless/cc
ARG BIN
COPY ./build/$BIN /$BIN
ENTRYPOINT ["/$BIN"]