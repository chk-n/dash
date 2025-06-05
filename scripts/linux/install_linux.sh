#!/bin/bash
# ONLY USE FOR CI/CD

make release-linux-x86_64

sudo mkdir /usr/local/dash
sudo tar -zxvf release/dash-${DASH_VERSION}-linux-x86_64.tar.gz -C /usr/local/dash
sudo ln -sf /usr/local/dash/bin/dsc /usr/local/bin/dsc

sudo chmod +x /usr/local/dash/bin/dsc
