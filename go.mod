module git.yunify.com/quanxiang/workflow

go 1.19

require (
	github.com/gin-gonic/gin v1.8.2
	github.com/go-kit/kit v0.12.0
	github.com/go-kit/log v0.2.1
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/quanxiang-cloud/cabin v0.0.6
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/xpsl/govaluate v3.1.0+incompatible

require (
	git.yunify.com/quanxiang/trigger v0.0.0-00010101000000-000000000000
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.11.1 // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/goccy/go-json v0.10.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.6 // indirect
	github.com/ugorji/go/codec v1.2.7 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.1 // indirect
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // indirect
	golang.org/x/net v0.4.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
	golang.org/x/text v0.5.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace git.yunify.com/quanxiang/trigger => ../trigger
