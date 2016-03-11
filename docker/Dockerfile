FROM debian:jessie
MAINTAINER Johannes Schüth <git@jotschi.de>

ARG BASEDIR
ENV DATA EMPTY

# Make sure the package repository is up to date.
RUN echo 'deb http://ftp.at.debian.org/debian jessie main contrib non-free' > /etc/apt/sources.list
RUN echo 'deb http://security.debian.org/ jessie/updates main contrib' >> /etc/apt/sources.list

RUN apt-get update ; apt-get -y upgrade ; apt-get install -y curl wget locales sudo sed

# Install Java 8
RUN mkdir -p /opt/java
ADD ./jdk-8u74-linux-x64.tar.gz /opt/java
RUN cd /opt/java ; ln -s jdk1.8.0_74 latest
RUN echo "PATH=$PATH:/opt/java/latest/bin" > /etc/profile.d/java.sh ; echo "JAVA_HOME=/opt/java/latest" >> /etc/profile.d/java.sh

# Setup locale/timezone
RUN echo "LC_ALL=en_US.UTF-8" > /etc/default/locale
RUN echo "LANG=en_US.UTF-8" >> /etc/default/locale
RUN echo "LANGUAGE=en_US.UTF-8" > /etc/default/locale

RUN sed -i  's/.*\(en_US.UTF-8.*\)/\1/' /etc/locale.gen
RUN locale-gen
RUN cp /usr/share/zoneinfo/Europe/Vienna /etc/localtime
RUN echo "Europe/Vienna" > /etc/timezone
COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
