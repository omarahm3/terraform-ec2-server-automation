#!/bin/bash
set -e

MAIN_DIR=/var/app
LOG_DIR=$MAIN_DIR/log

mkdir -p $MAIN_DIR
mkdir -p $LOG_DIR

sudo apt update -y
sudo apt install -y git python3.8 python3-pip

# Set ENV variables into ec2
echo "AWS_ACCESS_KEY_ID=${access_key}" >> /etc/environment
echo "AWS_SECRET_ACCESS_KEY=${secret_key}" >> /etc/environment
echo "AWS_DEFAULT_REGION=$(curl http://169.254.169.254/latest/dynamic/instance-identity/document|grep region|awk -F\" '{print $4}')" >> /etc/environment
echo "EC2_INSTANCE_ID=$(wget -q -O - http://169.254.169.254/latest/meta-data/instance-id)" >> /etc/environment
echo "PORT=80" >> /etc/environment
echo "FLASK_APP=server" >> /etc/environment

# Export every ENV variable immediately
# This should allow scripts to use variables like $AWS_DEFAULT_REGION immediately
for env in $(cat /etc/environment); do export $(echo $env | sed -e 's/"//g'); done

cd $MAIN_DIR

# Clone the server
git clone https://github.com/omarking05/simple-python-server.git server >> $LOG_DIR/mrg_git.log

cd server

pip3 install -r requirements.txt >> $LOG_DIR/mrg_pip.log

nohup python3 server.py >> $LOG_DIR/mrg_python.log &
