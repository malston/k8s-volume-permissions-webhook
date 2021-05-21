FROM alpine:latest

# install volume-permissions-container-injector binary
COPY build/_output/linux/bin/volume-permissions-container-injector /usr/local/bin/volume-permissions-container-injector

# set entrypoint
ENTRYPOINT ["/usr/local/bin/volume-permissions-container-injector"]

# switch to non-root user
USER 1001
