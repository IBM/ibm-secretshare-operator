FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ARG VCS_REF
ARG VCS_URL

LABEL org.label-schema.vendor="IBM" \
  org.label-schema.name="secretshare operator" \
  org.label-schema.description="An Ansible Operator to Share Secrets and ConfigMaps between namespaces" \
  org.label-schema.vcs-ref=$VCS_REF \
  org.label-schema.vcs-url=$VCS_URL \
  org.label-schema.license="Licensed Materials - Property of IBM" \
  org.label-schema.schema-version="1.0" \
  name="secretshare operator" \
  vendor="IBM" \
  description="An Ansible Operator to Share Secrets and ConfigMaps between namespaces" \
  summary="An Ansible Operator to Share Secrets and ConfigMaps between namespaces"

ENV OPERATOR=/usr/local/bin/secretshare \
    USER_UID=1001 \
    USER_NAME=secretshare

# install operator binary
COPY build/_output/bin/ibm-secretshare-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
