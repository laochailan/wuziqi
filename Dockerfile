FROM golang:1.22
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY templates.html /app/
COPY assets /app/assets
RUN CGO_ENABLED=0 GOOS=linux go build -o /wuziqi
EXPOSE 8080
CMD ["/wuziqi"]
