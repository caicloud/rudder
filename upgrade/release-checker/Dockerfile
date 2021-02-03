FROM cargo.caicloud.xyz/library/debian:stretch

COPY bin/release-checker /bin/release-checker
COPY templates/ /templates
ENV KUBECONFIG "/root/.kube/config"
ENV CHART_TEMPLATE_PATH "/templates"

WORKDIR /
ENTRYPOINT ["/bin/release-checker"]