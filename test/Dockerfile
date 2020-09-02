FROM registry.redhat.io/ubi8/ubi:8.2
ARG SAP_JDK_URL="https://github.com/SAP/SapMachine/releases/download/sapmachine-11.0.5/sapmachine-jdk-11.0.5-1.x86_64.rpm"
ARG HYBRIS_URL
ARG USERNAME
ARG PASSWORD
RUN curl -L -o /tmp/sapmachine-jdk.rpm $SAP_JDK_URL
RUN curl -u $USERNAME:$PASSWORD -v -k -o /tmp/hybris.tar.gz $HYBRIS_URL
