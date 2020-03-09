/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	github "github.com/google/go-github/v29/github"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	manifestupdaterkoyutaiov1alpha1 "manifest-updater/api/v1alpha1"
)

const (
	clonePath     = "/tmp/repository"
	defaultBranch = "master"
)

// UpdaterReconciler reconciles a Updater object
type UpdaterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=manifest-updater.koyuta.io.koyuta.io,resources=updaters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=manifest-updater.koyuta.io.koyuta.io,resources=updaters/status,verbs=get;update;patch

func (r *UpdaterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("updater", req.NamespacedName)

	updater := &manifestupdaterkoyutaiov1alpha1.Updater{}
	if err := r.Get(context.TODO(), req.NamespacedName, updater); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
	}

	setDefaultValuesIfNotPresent(updater)

	// TODO: Add supports for public cloud registry services.
	var registry = updater.Spec.Registry.DockerHub
	tags, err := fetchTagsFromDockerHub(registry)
	if err != nil {
		return ctrl.Result{}, err
	}

	var tag = tags[0] // use latest tag.
	if filter := updater.Spec.Registry.Filter; filter != "" {
		re, err := regexp.Compile(filter)
		if err != nil {
			return ctrl.Result{}, err
		}
		for _, t := range tags {
			if re.MatchString(t) {
				tag = t
				break
			}
		}
	}

	// TODO: Add the function that checks wheather the registry tag
	//       had been updated, using in-memory database like Redis.

	endpoint, err := transport.NewEndpoint(updater.Spec.Repository.Git)
	if err != nil {
		return ctrl.Result{}, err
	}
	path := strings.Split(endpoint.Path, "/")
	owner, reponame := path[0], path[1]

	repository, err := git.PlainCloneContext(context.TODO(), clonePath, false, &git.CloneOptions{
		URL:           updater.Spec.Repository.Git,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(updater.Spec.Repository.Branch),
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	worktree, err := repository.Worktree()
	if err != nil {
		return ctrl.Result{}, err
	}

	branch := fmt.Sprintf("feature/update-tag-%s:%s", registry, tag)
	if err := worktree.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: plumbing.NewBranchReferenceName(branch),
	}); err != nil {
		return ctrl.Result{}, err
	}

	var paths = []string{}
	if err = filepath.Walk(
		updater.Spec.Repository.Path,
		func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				paths = append(paths, path)
			}
			return nil
		}); err != nil {
		return ctrl.Result{}, err
	}

	re := regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(`%s:[^ ]+`, registry)))
	for _, p := range paths {
		content, err := ioutil.ReadFile(p)
		if err != nil {
			return ctrl.Result{}, err
		}
		replacedContent := re.ReplaceAll(content, []byte(tag))
		if err := ioutil.WriteFile(p, replacedContent, 0); err != nil {
			return ctrl.Result{}, err
		}
		if _, err := worktree.Add(p); err != nil {
			return ctrl.Result{}, err
		}
	}

	msg := ":up: Update image tag names from manifests"
	if _, err := worktree.Commit(msg, &git.CommitOptions{
		All: true,
		Committer: &object.Signature{
			Name: "manifest-updater",
		},
	}); err != nil {
		return ctrl.Result{}, err
	}

	if err := repository.PushContext(context.TODO(), &git.PushOptions{}); err != nil {
		return ctrl.Result{}, err
	}

	client := github.NewClient(nil)
	if _, _, err := client.PullRequests.Create(
		context.TODO(), owner, reponame, &github.NewPullRequest{
			Title:               github.String("Automaticaly update image tags"),
			Head:                github.String(branch),
			Base:                github.String(updater.Spec.Repository.Branch),
			Body:                github.String(""),
			MaintainerCanModify: github.Bool(true),
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func setDefaultValuesIfNotPresent(updater *manifestupdaterkoyutaiov1alpha1.Updater) {
	if updater.Spec.Repository.Branch == "" {
		updater.Spec.Repository.Branch = defaultBranch
	}
	if updater.Spec.Repository.Path == "" {
		updater.Spec.Repository.Path = filepath.Join(clonePath)
	}
}

func fetchTagsFromDockerHub(repository string) ([]string, error) {
	registry, err := name.NewRepository(repository)
	if err != nil {
		return []string{}, err
	}
	tags, err := remote.ListWithContext(context.TODO(), registry)
	if len(tags) == 0 {
		return []string{}, errors.New("No tags found")
	}
	return tags, err
}

func (r *UpdaterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&manifestupdaterkoyutaiov1alpha1.Updater{}).
		Complete(r)
}
