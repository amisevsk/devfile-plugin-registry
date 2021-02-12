FROM quay.io/libpod/golang:1.13 as builder

WORKDIR /build
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY ["main.go", "./"]
COPY ["meta_yaml", "./meta_yaml"]

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a \
    -o _output/bin/plugin-convert \
    -ldflags="-w -s" \
    main.go

# Use binary from above to rewrite metas in plugin registry
FROM quay.io/eclipse/che-plugin-registry:nightly AS plugin-registry
COPY --from=builder /build/_output/bin/plugin-convert /plugin-convert
COPY ["rewrite_metas.sh", "/rewrite_metas.sh"]
# Output is /build/v3...
RUN /rewrite_metas.sh 

# Build a "new" plugin registry
FROM docker.io/httpd:2.4.46-alpine
RUN apk add --no-cache bash && \
    # Allow htaccess
    sed -i 's|    AllowOverride None|    AllowOverride All|' /usr/local/apache2/conf/httpd.conf && \
    sed -i 's|Listen 80|Listen 8080|' /usr/local/apache2/conf/httpd.conf && \
    mkdir -p /var/www && ln -s /usr/local/apache2/htdocs /var/www/html && \
    chmod -R g+rwX /usr/local/apache2 && \
    echo "ServerName localhost" >> /usr/local/apache2/conf/httpd.conf && \
    apk add --no-cache coreutils
COPY --from=plugin-registry /usr/local/apache2/htdocs/.htaccess /usr/local/apache2/htdocs/.htaccess
COPY --from=plugin-registry /build/v3 /usr/local/apache2/htdocs/v3
RUN mkdir -p /usr/local/apache2/htdocs/v3/plugins/eclipse/che-theia/next
COPY ["manual/theia-next.yaml", "/usr/local/apache2/htdocs/v3/plugins/eclipse/che-theia/next/devfile.yaml"]
RUN echo "DirectoryIndex devfile.yaml" > /usr/local/apache2/htdocs/v3/plugins/.htaccess
ENTRYPOINT ["httpd-foreground"]
