FROM golang@sha256:62b42efa7bbd7efe429c43e4a1901f26fe3728b4603cb802248fff0a898b4825

# Install Zip
RUN apt-get update && apt-get upgrade -y && apt-get install -y zip

# Install Dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR /go/src/github.com/coinbase/odin

COPY Gopkg.lock Gopkg.toml ./

RUN dep ensure -vendor-only

COPY . .

RUN go build && go install

# builds lambda.zip
RUN ./scripts/build_lambda_zip
RUN shasum -a 256 lambda.zip | awk '{print $1}' > lambda.zip.sha256

RUN mv lambda.zip.sha256 lambda.zip /
RUN odin json > /state_machine.json

CMD ["odin"]
