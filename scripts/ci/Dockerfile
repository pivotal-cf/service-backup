FROM ubuntu:14.04

# Install packages
RUN \
      apt-get update && \
      apt-get -y install --fix-missing \
          curl \
          git \
          python \
          unzip \
          wget \
      && \
      apt-get clean

# Install AWS CLI
RUN \ 
      wget https://s3.amazonaws.com/aws-cli/awscli-bundle.zip && \
      unzip awscli-bundle.zip && \
      ./awscli-bundle/install \
            -i /usr/local/aws \
            -b /usr/local/bin/aws && \
      rm -rf awscli-bundle*

# Install go
RUN curl https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz | tar xvz -C /usr/local/

# Add golang environment variables
ENV GOPATH /go
ENV PATH $PATH:/usr/local/go/bin:$GOPATH/bin

# Create GOPATH
RUN \
      mkdir -p $GOPATH && \
      chmod 777 $GOPATH
