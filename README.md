# k8s-model

This is a set of Kubernetes (K8s) Custom Resource Types to encode the elements of EmELand model. The project also provides a controller, that makes K8s resources of these types available via a restful API.

In addition semantic checks are performed on the K8s resources and findings are also made available through that API.

## Release

Maintainers publish a release by pushing a semver git tag to `main`:

```bash
git tag v0.2.0
git push origin v0.2.0
```

CI then publishes:

- Operator image: `ghcr.io/emeland-io/modelsrv-k8s-sensor:0.2.0`
- CRD chart: `ghcr.io/emeland-io/charts/modelsrv-k8s-crd:0.2.0`
- Operator chart: `ghcr.io/emeland-io/charts/modelsrv-k8s-sensor:0.2.0`

Chart publish runs after the operator image workflow completes successfully.

### Install (sysadmin)

```bash
helm install modelsrv-k8s-crd oci://ghcr.io/emeland-io/charts/modelsrv-k8s-crd \
  --version 0.2.0 \
  --namespace emeland-system \
  --create-namespace

helm install modelsrv-k8s oci://ghcr.io/emeland-io/charts/modelsrv-k8s-sensor \
  --version 0.2.0 \
  --namespace emeland-system
```

See [charts/modelsrv-k8s-crd/README.md](charts/modelsrv-k8s-crd/README.md) and [charts/modelsrv-k8s-sensor/README.md](charts/modelsrv-k8s-sensor/README.md) for details.

## Authors

* [cypherfox](https://gitlab.com/cypherfox)

## License

This project is licensed under the Apache License 2.0.

You can find a copy of the license in the file [LICENSE](LICENSE)

## TODO

### Badges

* [ ] TODO

On some READMEs, you may see small images that convey metadata, such as whether or not all the tests are passing for the project. You can use Shields to add some to your README. Many services also have instructions for adding a badge.

### Visuals

* [ ] TODO

Depending on what you are making, it can be a good idea to include screenshots or even a video (you'll frequently see GIFs rather than actual videos). Tools like ttygif can help, but check out Asciinema for a more sophisticated method.

### Installation

* [ ] TODO

Within a particular ecosystem, there may be a common way of installing things, such as using Yarn, NuGet, or Homebrew. However, consider the possibility that whoever is reading your README is a novice and would like more guidance. Listing specific steps helps remove ambiguity and gets people to using your project as quickly as possible. If it only runs in a specific context like a particular programming language version or operating system or has dependencies that have to be installed manually, also add a Requirements subsection.

### Usage

* [ ] TODO

Use examples liberally, and show the expected output if you can. It's helpful to have inline the smallest example of usage that you can demonstrate, while providing links to more sophisticated examples if they are too long to reasonably include in the README.

### Support

* [ ] TODO

Tell people where they can go to for help. It can be any combination of an issue tracker, a chat room, an email address, etc.

### Roadmap

* [ ] TODO

If you have ideas for releases in the future, it is a good idea to list them in the README.

### Contributing

* [ ] TODO

State if you are open to contributions and what your requirements are for accepting them.

For people who want to make changes to your project, it's helpful to have some documentation on how to get started. Perhaps there is a script that they should run or some environment variables that they need to set. Make these steps explicit. These instructions could also be useful to your future self.

You can also document commands to lint the code or run tests. These steps help to ensure high code quality and reduce the likelihood that the changes inadvertently break something. Having instructions for running tests is especially helpful if it requires external setup, such as starting a Selenium server for testing in a browser.

### Project status

* [ ] TODO

If you have run out of energy or time for your project, put a note at the top of the README saying that development has slowed down or stopped completely. Someone may choose to fork your project or volunteer to step in as a maintainer or owner, allowing your project to keep going. You can also make an explicit request for maintainers.
