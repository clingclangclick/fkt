# FKT

FLuxCD Kind of Templater

[![Release](https://github.com/clingclangclick/fkt/actions/workflows/release.yml/badge.svg)](https://github.com/clingclangclick/fkt/actions/workflows/release.yml)

## Usage

```shell
Usage: fkt

FluxCD Kind of Templater.

Flags:
  -h, --help                        Show context-sensitive help.
  -f, --config-file=STRING          YAML configuration file ($CONFIG_FILE)
  -b, --base-directory="."          Sources and overlays base directory ($BASE_DIRECTORY)
  -d, --dry-run                     Dry run and return error if changes are needed ($DRY_RUN)
  -v, --validate                    Validate configuration ($VALIDATE)
  -l, --logging.level="default"     Log level ($LOG_LEVEL)
  -o, --logging.file=STRING         Log file ($LOG_FILE)
  -t, --logging.format="default"    Log format ($LOG_FORMAT)
```

### Example

``` yaml
---
settings:
  directories:
    templates: templates       # resource templates path
    targets: clusters          # output parent target path
  delimiters:
    left: '[[['                # custom left delimiter
    right: ']]]'               # custom right delimiter
  log:
    level: debug               # log level, one of none, trace, debug, info, warn, error. Default is none
    format: json               # log format, one of console, json. Default is console
    file: 'log.txt'            # log file, none infers stdout
anchors:                       # re-usable anchors
  flux-system: &flux-system
    flux-system:
      managed: false             # FluxCD system resources are managed by flux
  flux-sops-key: &flux-sops-key  # Kustomization patch for flux secrets
    patch: |
        apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
        kind: Kustomization
        metadata:
          name: all
        spec: 
          decryption:
            provider: sops
            secretRef:
              name: sops-key
    target:
      kind: Kustomization
      labelSelector: app.kubernetes.io/sops=enabled
secrets:
  file: secrets.yaml           # SOPS age encrypted secrets values, requires env variables to decode, optional
values:                        # global values
  global_key: global_value
  global_array_keys:
    - global_array_key: global_array_value
    - global_array_property:
        global_array_property_key: global_array_property_value
  global_slices:
    - global_slice_one
    - global_slice_two
clusters:
  <path>:                      # cluster path
    managed: true              # prune cluster output directory, manage top-level kustomize.yaml file, available as `.Cluster.managed`
    kustomization:             # top-level kustomization options
      commonAnnotations:       # Kustomization annotations, available as `.Cluster.commonAnnotations`
        platform: platform
        name: name
        region: region
      patches:                 # Kustomization patches
      - *flux-sops-key
    annotations:               # annotations for cluster, added into kustomize file, available as `.Cluster.annotations`
      key: value               # cluster k/v annotation example, available as `.Cluster.annotations.key`
      name: name               # annotation name defaults to path stem if unset
    values:                    # cluster level values, supercedes global values
      cluster_key: cluster_value
      cluster_array_keys:
        - cluster_array_slice: cluster_array_value
      cluster_slices:
        - cluster_slice_one
        - cluster_slice_two
    age_public_key: <key>     # Age public key for secrets encryption
    resources:                # resources to include in cluster output
      <<: [*flux-system]      # Include flux-system resource anchor
      example:                # resource name
        template: ex          # resource template name, default is resource name, accessed as `.Reource.template`
        managed: true         # managed reource, will not remove overlay if cluster is managed and resource non-existent
        namespace: example    # optional namespace, default to resource name, accessed as `.Resource.namespace`
        values:               # values, overrides cluster and global leval, accessed as `.Values.<map name>`
          data: test-date     #   `.Values.data`
```

## Cluster paths

Cluster paths are unique within the `clusters` mapping and are paths that render
output in the target directory.

### Managed cluster

A managed cluster resets the cluster directory when ran, EXCEPT if resource is
unmanaged, i.e. `<reource>.Managed` is `false`. In this case, the unmanaged
resource path for the cluster are:

* Not removed
* If a `kustomization.yaml` or `kustomization.yml` exists, the cluster
  `kustomization.yaml` will include the unmanaged resource.

Managed resource functionality is used primarily to support FluxCD cluster
bootstrap of a managed cluster, but other resources can be marked to not
be removed in a cluster target output.

## Values

Values are accessed as `.Values.<property>` Properties are replaced if a
lower-level setting updates the property. Values are not merged.

Evaluation order:

* Global
* Cluster
* Reource

### Global values

Global valuse are in the upper-level schema.

### Cluster values

Access as `.Cluster.<property>`

Properties:

* `commonAnnotations`: Kustomization CommonAnnotations
* `managed`: Managed boolean
* `path`: Cluster path

### Reource values

Access as `.Resource.<property>`

Properties:

* `name`: Resource name
* `namespace`: Resource namespace
* `template`: Resource template path, allows for re-using sources

### Sprig templating functions

Templating uses [sprig](http://masterminds.github.io/sprig/) functions.

For example, values as:

```yaml
array_keys:
  - array_slice: array_value
```

Templated as:

```yaml
keys: "[[[ keys .Values | join "," ]]]"
array_keys: [[[ first .Values.array_keys | values | first ]]]
```

Become:

```yaml
keys: "cluster_array_keys"
array_keys: cluster_array_value
```

## Secrets

Simple support for SOPS age encrypted secrets is supported. Using the configuration:

```yaml
secrets:
  file: secrets.yaml
```

An age-encrypted SOPS file `secrets.yaml` is decoded and added to a `Secrets`
value for templating K8S templates with `Kind: Secret`. Clusters are assigned
an age public key:

```yaml
clusters:
  <cluster_path>:
    age_public_key: <public key>
```

The environmental variable SOPS_AGE_KEY_FILE or SOPS_AGE_KEY must be set
or the secrets file cannot be decrypted and `fkt` will error out if a
secrets file is supplied in the configuration.

Secrets are available in `Secret` K8S template files as `.Secrets`,
so that a `secrets.yaml` file:

```yaml
secret: value
```

The value of `secret` is `.Secrets.secret`.
The secrets are base64 encoded in the template:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: [[[ .Resource.name ]]]
  namespace: [[[ .Resource.namespace ]]]
data:
  secret: [[[ .Secrets.secret | b64enc ]]]
```

## Generated Kustomization

Kustomization files are generated for each target path, which can be
set in the cluster configuration.



## Bootstrapping FluxCD

Include a `flux-system` anchor in the YAML configuration

```yaml
  flux-system: &flux-system
    flux-system:
      managed: false
```

Add the anchor to the cluster resources property:

```yaml
    resources:
      <<: [*flux-system]
```

After running `fkt`, no upper-level Kustomization will exist in the cluster path.
It is safe to commit this configuration into the config repo.

In the config repo, bootstrap fluxcd into the cluster path. FluxCD will install
on the cluster and update the repository to include a `flux-system` path in
the cluster overlay. Pull the changes, as Flux may have altered the default
branch. Running `fkt` again will add the `flux-system` Kustomizations to the
cluster Kustomization.

## YAML spec

### Config type

```golang
type Config struct {
  Settings *Settings           `yaml:"settings"`
  Values   Values              `yaml:"values,flow"`
  Clusters map[string]*Cluster `yaml:"clusters"`
  Secrets  struct {
    SecretsFile string  `yaml:"file"`
  } `yaml:"secrets"`
}
```

### Settings type

```golang
type Settings struct {
  Delimiters struct {
    Left  string `yaml:"left"`
    Right string `yaml:"right"`
  } `yaml:"delimiters"`
  Directories struct {
    Templates     string `yaml:"templates"`
    Targets       string `yaml:"targets"`
    baseDirectory string
  } `yaml:"directories"`
  DryRun    bool       `yaml:"dry_run"`
  LogConfig *LogConfig `yaml:"log"`
}
```

#### LogConfig type

```golang
type LogConfig struct {
  Level  LogLevel `yaml:"level"`  // One of none (panic), trace, debug, info, error
  File   string   `yaml:"file"`   // Default stdout
  Format string   `yaml:"format"` // One of console, json. Default console
}
```

### Cluster type

```golang
type Cluster struct {
  Kustomization *Kustomization       `yaml:"kustomization,flow"`
  Managed       *bool                `yaml:"managed"`
  Values        *Values              `yaml:"values,flow"`
  Resources     map[string]*Resource `yaml:"resources,flow"`
  AgePublicKey  string               `yaml:"age_public_key"`
  path          *string
}
```

### Kustomization type

```golang
type Kustomization struct {
  APIVersion        string            `yaml:"apiVersion"`
  Kind              string            `yaml:"kind"`
  Resources         []string          `yaml:"resources"`
  CommonAnnotations map[string]string `yaml:"commonAnnotations"`
  Patches           []interface{}     `yaml:"patches"`
}
```

### Resource type

```golang
type Resource struct {
  Template  *string `yaml:"template"`
  Namespace *string `yaml:"namespace"`
  Values    Values  `yaml:"values,flow"`
  Managed   *bool   `yaml:"managed"`
  Name      string
}
```

## Pre-Commit config

A pre-commit config can be used to automatically update the cluster overlays
after changing the configuration.

```yaml
- repo: git@<repo>
  rev: <tag>
  hooks:
  - id: fkt
    always_run: true
    args: [-l, debug]
```

## GitHub action

A GitHub `action.yml` is included to verify that the supplied configuration
would be unchanged to ensure the configuration output is consistent with
the overlay contents for all clusters.

Arguments:

* `base-directory`: Default to `.`
* `config-file`: Default to `config.yaml`
* `log-level`: Default to `warn`

### FKT hosted as public repository or Enterprise private repository

```yaml
    - name: Validate Overlay
      uses: <org>/<repo>@v0
```

### FKT hosted as private repository, non-Enterprise

A PAT is required to load the repo into the GH working directory.

```yaml
    - name: Checkout Validate Overlay Action
      uses: actions/checkout@v4
      with:
        repository: <org><repo>
        ref: <version>
        token: ${{ secrets.INTERNAL_PAT }}
        path: ./.github/actions/fkt
    - name: Validate Overlay
      id: validate-overlay
      uses: ./.github/actions/fkt
      with:
        log-level: debug
```
