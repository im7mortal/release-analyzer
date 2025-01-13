#!/bin/bash

docker build -t im7mortal/release-analyzer:latest . && docker run -p 8080:8080 im7mortal/release-analyzer:latest
