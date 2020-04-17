build:
		go get github.com/go-kit/kit/log
		go get github.com/go-kit/kit/log/level
		go get github.com/prometheus/common/version
		go get github.com/prometheus/client_golang/prometheus
		go get github.com/prometheus/client_golang/prometheus/promhttp
		go get gopkg.in/alecthomas/kingpin.v2
		go get gopkg.in/yaml.v2
		go build

run:
		go run main.go
