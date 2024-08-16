FROM node:22.2.0-alpine as front
WORKDIR /app
COPY front/package*.json ./
RUN npm install
COPY front/. .
RUN npm run generate
COPY front/replace_underscore.sh ./
RUN chmod +x /app/replace_underscore.sh
RUN apk add --no-cache dos2unix && dos2unix /app/replace_underscore.sh
RUN /app/replace_underscore.sh

FROM golang:1.22.5-alpine as backend
ENV CGO_ENABLED 1
RUN apk add build-base
WORKDIR /app
COPY backend/go.mod .
COPY backend/go.sum .
RUN go mod download
RUN go mod tidy
COPY backend/. .
COPY --from=front /app/.output/public /app/app/public
RUN apk update --no-cache && apk add --no-cache tzdata
RUN go build -tags prod -ldflags="-s -w" -o /app/moments

FROM alpine
ARG VERSION
RUN apk update --no-cache && apk add --no-cache ca-certificates
COPY --from=backend /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ Asia/Shanghai
WORKDIR /app/data
ENV VERSION $VERSION
COPY --from=backend /app/moments /app/moments
ENV PORT 3000
EXPOSE 3000
RUN chmod +x /app/moments
CMD ["/app/moments"]