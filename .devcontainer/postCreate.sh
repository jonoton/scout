#!/bin/bash

# Install Jekyll dependencies for docs/
gem install jekyll bundler
cd docs
bundle install
cd ..
