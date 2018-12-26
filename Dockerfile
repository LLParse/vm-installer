FROM scratch
COPY release/vm-installer /
ENTRYPOINT ["/vm-installer"]
