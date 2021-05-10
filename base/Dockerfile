## Stage 1 : download the SAP JDK and Hybris package
FROM registry.redhat.io/ubi8/ubi:8.2 as downloader
RUN curl -u $USERNAME:$PASSWORD -L -o /tmp/sapmachine-jdk.rpm $SAP_JDK_URL
RUN curl -u $USERNAME:$PASSWORD -v -k -o /tmp/hybris.tar.gz $HYBRIS_URL
RUN tar -xzf /tmp/hybris.tar.gz -C /tmp

## Stage 2 : build the hybris base image
FROM registry.redhat.io/ubi8/ubi:8.2

LABEL io.openshift.s2i.destination="/tmp"  \
      io.openshift.s2i.scripts-url="image:///usr/local/s2i" \
      maintainer="Red Hat SAP Community of Practice"

ENV SAPMACHINE_FILE_NAME='sapmachine-jdk.rpm' \
    HYBRIS_HOME='/opt/hybris'

RUN INSTALL_PKGS="gettext procps-ng hostname net-tools lsof" && \
   yum install -y $INSTALL_PKGS && \
   rpm -V $INSTALL_PKGS && \
   yum clean all

COPY --from=downloader /tmp/hybris $HYBRIS_HOME
COPY --from=downloader /tmp/$SAPMACHINE_FILE_NAME /tmp

RUN yum install -y /tmp/$SAPMACHINE_FILE_NAME && \
    rm /tmp/$SAPMACHINE_FILE_NAME && \
    cp -R $HYBRIS_HOME/bin/platform/resources/configtemplates/production $HYBRIS_HOME/config && \
    useradd -u 1001 -r -g 0 -d $HYBRIS_HOME -s /sbin/nologin -c "Hybris User" hybris && \
    chmod -R g+rwX $HYBRIS_HOME && \
    chown -R 1001:root $HYBRIS_HOME

COPY s2i /usr/local/s2i

RUN chmod -R u+rwx /usr/local/s2i && \
    chmod -R g+rwx /usr/local/s2i && \
    chown -R 1001:root /usr/local/s2i

EXPOSE 9001
EXPOSE 9002

USER 1001