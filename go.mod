module github.com/A-way/away

go 1.15

replace golang.org/x/crypto v0.0.0-20180119165957-a66000089151 => github.com/golang/crypto v0.0.0-20180119165957-a66000089151

require (
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/onsi/gomega v1.10.4 // indirect
	github.com/sirupsen/logrus v1.0.5-0.20180213143110-8c0189d9f6bb
	github.com/stretchr/testify v1.6.1 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
)

replace golang.org/x/net v0.0.0-20180112015858-5ccada7d0a7b => github.com/golang/net v0.0.0-20180112015858-5ccada7d0a7b

replace golang.org/x/sys v0.0.0-20180202135801-37707fdb30a5 => github.com/golang/sys v0.0.0-20180202135801-37707fdb30a5
