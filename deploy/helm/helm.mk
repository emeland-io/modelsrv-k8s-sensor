HELM                        ?= helm
HELM_PATH 		            ?= deploy/helm
HELM_OUTPUT_DIR             ?= tmp/helm
HELM_CHANNEL                ?= dev
HELM_REGISTRY               ?= 
HELM_CHART_NAME             ?= emeland-k8s
HELM_CRD_CHART_NAME         ?= emeland-k8s-crd
HELM_VERSION                ?= 0.1.0
HELM_PUBLISH_URL            ?= https://gitlab.opencode.de/api/v4/projects/4311/packages/helm/api/$(HELM_CHANNEL)/charts
HELM_REGISTRY_ALIAS         ?= opencode
HELM_RELEASE                ?= emeland-k8s
HELM_CRD_RELEASE            ?= emeland-k8s-crd
HELM_TEMPLATES_DIR          ?= ./deploy/helm/emeland-k8s/templates
HELM_CRD_TEMPLATES_DIR      ?= ./deploy/helm/emeland-k8s/templates
HELM_VALUES_FILE            ?= examples/00-dev-values.yaml
CONFIG_DIR                  ?= ./config
OAPI_DIR					?= ./api/oapi/emeland-k8s
OAPI_FILE                   ?= EmergingEnterpriseLandscape-0.1.0-oapi3.0.3.yml
KUBE_NAMESPACE              ?= emeland-k8s-system

.PHONY: helm-clean helm-dep helm-lint helm-install-from-repo helm-uninstall helm-template helm-add-opencode helm-docs helm-publish

##@ Helm

helm-clean: ## clean up templated helm charts
	@rm -Rf $(HELM_OUTPUT_DIR)

helm-dep-%: ## update helm dependencies
	@$(HELM) dep update $(HELM_PATH)/$*

helm-lint: $(HELM_TEMPLATES_DIR)/rbac.yaml $(HELM_CRD_TEMPLATES_DIR)/crds.yaml $(HELM_CRD_TEMPLATES_DIR)/rbac.yaml $(HELM_TEMPLATES_DIR)/api-spec-configmap.yaml ## lint helm chart
	@$(HELM) lint $(HELM_PATH)/$(HELM_CHART_NAME)
	@$(HELM) lint $(HELM_PATH)/$(HELM_CRD_CHART_NAME)

helm-install-from-repo: ## install helm chart from build artifact
	@$(HELM) repo update
	@$(HELM) upgrade $(HELM_RELEASE) $(HELM_REGISTRY_ALIAS)/$(HELM_CHART_NAME) --install --namespace $(KUBE_NAMESPACE) --version $(VERSION) --values $(HELM_VALUES_FILE) --skip-crds

helm-install: 
	$(HELM) upgrade $(HELM_RELEASE) ./$(HELM_CHART_NAME)-$(HELM_VERSION).tgz --install \
	  --namespace $(KUBE_NAMESPACE) --create-namespace \
	  --set image.repository=$(IMAGE_REPO) --set image.tag=$(IMAGE_VERSION) --devel --debug

helm-install-crds: 
	$(HELM) upgrade $(HELM_CRD_RELEASE) ./$(HELM_CRD_CHART_NAME)-$(HELM_VERSION).tgz --install --devel --debug

helm-uninstall: ## uninstall helm chart
	@$(HELM) uninstall $(HELM_RELEASE) --namespace $(KUBE_NAMESPACE)

helm-uninstall-crds: ## uninstall helm CRD chart
	@$(HELM) uninstall $(HELM_CRD_RELEASE)

helm-template-%: helm-clean ## template helm chart
	@mkdir -p $(HELM_OUTPUT_DIR)
	@$(HELM) template $(HELM_RELEASE) $(HELM_PATH)/$* --namespace $(KUBE_NAMESPACE) --values $(HELM_PATH)/$*/$(HELM_VALUES_FILE) --output-dir $(HELM_OUTPUT_DIR) --include-crds --debug
	@echo "ATTENTION:"
	@echo "If you want to have the latest dependencies (e.g. gateway chart changes)"
	@echo "execute the following command prior to the current command:"
	@echo "$$ $(MAKE) helm-dep-$*"
	@echo

helm-add-opencode: ## add opencode helm chart repo
	@$(HELM) repo add $(HELM_REGISTRY_ALIAS) "$(HELM_REGISTRY)"

helm-set-version-all:
	@find $(HELM_PATH) -name 'Chart.yaml' -exec $(YQ) e --inplace '.version = "$(VERSION)"' {} \;
	@find $(HELM_PATH) -name 'Chart.yaml' -exec $(YQ) e --inplace '.appVersion = "$(VERSION)"' {} \;
	@find $(HELM_PATH) -name 'Chart.yaml' -exec $(YQ) e --inplace '(.dependencies.[].version | select(. == "0.0.1-local")) |= "$(VERSION)"' {} \;

helm-docs: ## update the auto generated docs of all helm charts
	@docker run --rm --volume "$(HELM_PATH)/$(HELM_CHART_NAME):/helm-docs" -u $(shell id -u) jnorwood/helm-docs:v1.4.0 --template-files=./README.md.gotmpl
	@docker run --rm --volume "$(HELM_PATH)/$(HELM_CRD_CHART_NAME):/helm-docs" -u $(shell id -u) jnorwood/helm-docs:v1.4.0 --template-files=./README.md.gotmpl

helm-publish: helm-lint
	$(HELM) package $(HELM_PATH)/$(HELM_CHART_NAME) 
	$(HELM) package $(HELM_PATH)/$(HELM_CRD_CHART_NAME)
	source .gitlab/gitlab.opencode.de-deploy-token.sh && echo user: $$USERNAME && \
	  curl --request POST --form 'chart=@$(HELM_CHART_NAME)-$(HELM_VERSION).tgz' \
        --user $$USERNAME:$$PASSWORD $(HELM_PUBLISH_URL) && \
	  curl --request POST --form 'chart=@$(HELM_CRD_CHART_NAME)-$(HELM_VERSION).tgz' \
        --user $$USERNAME:$$PASSWORD $(HELM_PUBLISH_URL)

helm-publish-CI: helm-lint
	$(HELM) package $(HELM_PATH)/$(HELM_CHART_NAME) 
	$(HELM) package $(HELM_PATH)/$(HELM_CRD_CHART_NAME)
	curl --request POST --form 'chart=@$(HELM_CHART_NAME)-$(HELM_VERSION).tgz' \
        --user gitlab-ci-token:$(CI_JOB_TOKEN) $(HELM_PUBLISH_URL) && \
	  curl --request POST --form 'chart=@$(HELM_CRD_CHART_NAME)-$(HELM_VERSION).tgz' \
        --user gitlab-ci-token:$(CI_JOB_TOKEN) $(HELM_PUBLISH_URL)


# set up RBAC resource files as template
#  - remove 'managed-by' label
#  - make namespace configurable through .Release.Namespace
#  - put into single rbac.yaml template file
# TODO: metrics reader and metrics auth
$(HELM_TEMPLATES_DIR)/rbac.yaml: $(CONFIG_DIR)/rbac/leader_election_role.yaml $(CONFIG_DIR)/rbac/*_binding.yaml
	@printf "# DO NOT EDIT MANUALLY!\n# This file is generated from the contents of the config/rbac folder by the make target 'helm-publish'. \n\n" > $@
	@for file in $^ ; do sed '/app.kubernetes.io\/managed-by:/d' $$file | sed 's/namespace: system/namespace: \{\{ \.Release\.Namespace \}\}/' >> $@ ; printf "\n---\n\n"  >> $@ ; done

# create configmap from the OAPI 3.0 file, so that helm can fix the url of the server at deploy time of the helm chart
$(HELM_TEMPLATES_DIR)/api-spec-configmap.yaml: $(OAPI_DIR)/$(OAPI_FILE) $(HELM_TEMPLATES_DIR)/_helpers.tpl
	@printf "# DO NOT EDIT MANUALLY!\n# This file is generated from the contents of the OAPI file in the API folder by the make 'helm-publish'. \n\n" > $@
	@printf "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: oapi-spec\nimmutable: true\ndata:\n  swagger-config.yaml: |\n" >> $@
	@sed 's+- url: https://api.server.test/v1+\{\{- include "emeland-k8s.swaggerHostURL" \. \}\}+' $^ | sed 's/^/    /' >> $@


# set up CRD resources files as template
#  - put into single crds.yaml template file
$(HELM_CRD_TEMPLATES_DIR)/crds.yaml: $(CONFIG_DIR)/crd/bases/*.yaml
	@printf "# DO NOT EDIT MANUALLY!\n# This file is generated from the contents of the config/crd/bases folder by the make target 'helm-publish'. \n\n" > $@
	@for file in $^ ; do cat $$file >> $@ ; printf "\n---\n\n" >> $@ ; done

# set up RBAC resources related to the CRDs as template
#  - remove 'managed-by' label
#  - put into single rbac.yaml template file
$(HELM_CRD_TEMPLATES_DIR)/rbac.yaml: $(CONFIG_DIR)/rbac/*_editor_role.yaml $(CONFIG_DIR)/rbac/*_viewer_role.yaml $(CONFIG_DIR)/rbac/role.yaml
	@printf "# DO NOT EDIT MANUALLY!\n# This file is generated from the contents of the config/rbac folder by the make target 'helm-publish'. \n\n" > $@
	@for file in $^ ; do sed '/app.kubernetes.io\/managed-by:/d' $$file >> $@ ; printf "\n---\n\n" >> $@ ; done

