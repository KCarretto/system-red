FROM ubuntu:16.04
COPY build/system_red /sbin/init
ENTRYPOINT [ "/sbin/init" ]
