FROM golang@sha256:6486ea568f95953b86c9687c1e656f4297d9b844481e645a00c0602f26fee136

# Install Zip
RUN apt-get update && apt-get upgrade -y && apt-get install -y zip

WORKDIR /go/src/github.com/coinbase/odin

ENV GO111MODULE on
ENV GOPATH /go

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build && go install

# builds lambda.zip
RUN ./scripts/build_lambda_zip
RUN shasum -a 256 lambda.zip | awk '{print $1}' > lambda.zip.sha256

RUN mv lambda.zip.sha256 lambda.zip /
RUN odin json > /state_machine.json

CMD ["odin"]
