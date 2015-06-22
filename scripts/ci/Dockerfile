FROM ubuntu:14.04

# Install packages
RUN \
      apt-get update && \
      apt-get -y install --fix-missing \
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

