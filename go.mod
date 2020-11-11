module sigs.k8s.io/instrumentation-tools

go 1.14

replace (
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a // pinned to release-branch.go1.13
	golang.org/x/tools => golang.org/x/tools v0.0.0-20190821162956-65e3620a7ae7 // pinned to release-branch.go1.13
	k8s.io/api => k8s.io/api v0.0.0-20200307122242-510bcd53e1cf
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20200307122051-2b7fa1cb5395
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20200307124427-99b536c4b997
	k8s.io/client-go => k8s.io/client-go v0.0.0-20200307122516-5194bac86967
)

require (
	github.com/c-bata/go-prompt v0.2.4-0.20200321140817-d043be076398
	github.com/fatih/color v1.9.0
	github.com/gdamore/tcell v1.3.0
	github.com/golang/protobuf v1.3.3
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e
	github.com/mattn/go-runewidth v0.0.9
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.7.0
	github.com/pkg/term v0.0.0-20200520122047-c3ffed290a03 // indirect
	github.com/prometheus/client_golang v1.2.0
	github.com/prometheus/prometheus v1.8.2-0.20200213233353-b90be6f32a33
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/apimachinery v0.0.0-20200307122051-2b7fa1cb5395
	k8s.io/cli-runtime v0.0.0-00010101000000-000000000000
	k8s.io/client-go v0.0.0-20200307122516-5194bac86967
)
