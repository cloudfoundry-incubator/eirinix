# eirinix-helm-release pipeline

[This pipeline] builds the SUSE Eirini extensions docker images, as well as
building the SUSE Eirini extensions helm chart.  They are for use with [kubecf].

[This pipeline]: https://concourse.suse.dev/teams/main/pipelines/eirinix
[kubecf]: https://code.cloudfoundry.org/kubecf

## Deploying the pipeline

### Prerequisites

Deploying this pipeline requires a [concourse] installation, as well as having
[gomplate] available locally.

[concourse]: https:/concourse-ci.org/
[gomplate]: https://gomplate.ca/

The concourse [credential manager] is also required to have the
`docker-username` and `docker-password` credentials available.

[credential manager]: https://concourse-ci.org/creds.html

### Deploy on concourse

Deploy the pipeline using the `fly` script in this directory:

```bash
./fly -t target set-pipeline
```

### Pipeline development

If you wish to deploy a copy of this pipeline for development (without
publishing) artifacts to the official places), create a `configl.yaml` file
based on `config.yaml.sample` and deploy normally.
