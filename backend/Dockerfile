FROM ubuntu:latest

RUN apt-get update
RUN apt-get install -y wget git gcc

RUN wget -P /tmp https://dl.google.com/go/go1.12.5.linux-amd64.tar.gz

RUN tar -C /usr/local -xzf /tmp/go1.12.5.linux-amd64.tar.gz
RUN rm /tmp/go1.12.5.linux-amd64.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# プロジェクトルートに移動する
WORKDIR /go/src/portfolio/saborie/backend

# なんでこれがいるのか？？
COPY . .

RUN go get -d -v ./...
# 大体、neo4jドライバーを入れるための設定
RUN apt install -y libssl1.0.0
RUN wget https://github.com/neo4j-drivers/seabolt/releases/download/v1.7.4/seabolt-1.7.4-Linux-ubuntu-18.04.deb
RUN dpkg -i seabolt-1.7.4-Linux-ubuntu-18.04.deb
RUN rm seabolt-1.7.4-Linux-ubuntu-18.04.deb
RUN export PKG_CONFIG_PATH=/seabolt/build/dist/share/pkgconfig
RUN export LD_LIBRARY_PATH=/seabolt/build/dist/lib64
RUN export C_INCLUDE_PATH=/seabolt/build/dist/include
RUN apt install -y pkg-config
RUN apt install -y build-essential cmake

#CMD [ "go", "run", "main.go" ]