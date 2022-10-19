FROM node:18.0-alpine

RUN apk add --no-cache bash

# Create app directory
WORKDIR /app

COPY package.json .
RUN npm install --quiet

COPY src ./src