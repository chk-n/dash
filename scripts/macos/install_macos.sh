#!/bin/bash
# ONLY USE FOR CI/CD

profile=~/.zshrc

make release-darwin-aarch64
sudo mkdir /usr/local/dash
sudo installer -pkg release/dash-${DASH_VERSION}-darwin-aarch64.pkg -target /

sudo chown -R $(whoami) /usr/local/dash
chmod -R 755 /usr/local/dash

sudo ln -sf /usr/local/dash/bin/dsc /usr/local/bin/dsc
