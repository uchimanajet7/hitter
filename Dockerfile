# We also expect it to run directly on the EC2
FROM amazonlinux:2

# https://nodejs.org/en/about/releases/
ARG NODE_VER="14.x"

# https://golang.org/dl/
ARG GO_VER="1.15.2"

RUN amazon-linux-extras install -y epel \
    && yum update --obsoletes -y \
    && yum install -y glibc-langpack-ja \
    && curl --silent --location https://rpm.nodesource.com/setup_${NODE_VER} | bash - \
    && yum install nodejs -y \
    && yum install python3 -y \
    # && amazon-linux-extras install python3.8 -y \
    && yum install git tar iproute make -y \
    && curl -L -O https://dl.google.com/go/go${GO_VER}.linux-amd64.tar.gz \
    && tar -C /usr/local -xvzf go${GO_VER}.linux-amd64.tar.gz \
    && rm go${GO_VER}.linux-amd64.tar.gz \
    && yum clean all \
    && rm -rf /var/cache/yum

ENV TERM vt100
ENV PYTHONUNBUFFERED 1
ENV LANG ja_JP.utf8
ENV LC_ALL ja_JP.utf8
ENV PATH $PATH:/usr/local/go/bin

WORKDIR /hitter
ADD . /hitter

RUN unlink /etc/localtime \
    && ln -s /usr/share/zoneinfo/Japan /etc/localtime

RUN npm install -g npm \
    && npm install -g aws-cdk \
    && pip3 install pip --upgrade \
    && pip3 install awscli --upgrade \
    && pip3 install aws-cdk.core --upgrade \
    && echo alias ll="'ls -la'" > ~/.bashrc \
    && echo complete -C "'/usr/local/bin/aws_completer'" aws >> ~/.bashrc 

# CMD ["/bin/bash"]
