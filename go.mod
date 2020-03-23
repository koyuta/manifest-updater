module manifest-updater

go 1.13

require (
	github.com/go-chi/chi v4.0.3+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/google/go-containerregistry v0.0.0-20200310013544-4fe717a9b4cb
	github.com/google/go-github v17.0.0+incompatible
	github.com/koyuta/manifest-updater v0.1.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/urfave/cli/v2 v2.2.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	gopkg.in/src-d/go-git.v4 v4.13.1
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)
