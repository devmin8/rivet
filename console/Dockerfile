# syntax=docker/dockerfile:1.7

FROM oven/bun:1.3.1-alpine AS build

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .
ARG VITE_RIVET_API_URL=/api/v1
ENV VITE_RIVET_API_URL=$VITE_RIVET_API_URL
RUN bun run build-only

FROM nginx:1.29-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
