FROM ubuntu:16.04
COPY build/system_red /bin/system_red
CMD ["/bin/system_red"]
