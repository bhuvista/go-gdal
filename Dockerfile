FROM golang:latest
RUN apt install gcc
WORKDIR /app
COPY . /app/