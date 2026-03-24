# `tfx.yaml` Schema

`cmd/tfx` can run a real project pipeline from a local YAML file.

It looks for these filenames, walking upward from the current working directory:

- `tfx.yaml`
- `tfx.yml`
- `.tfx.yaml`
- `.tfx.yml`

If no config is found, `cmd/tfx` falls back to a local pipeline:

- nearest `go.mod`: `go test ./...`, `go build ./...`, `go vet ./...`
- otherwise: `pwd`, `ls`

## Two Modes

`cmd/tfx` supports two config styles:

1. Single pipeline via top-level `steps`
2. Multiple named pipelines via `flows`

`profiles` is accepted as an alias for `flows`, and `default_profile` is accepted as an alias for `default_flow`.

## Top-Level Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `project` | `string` | no | Project label shown in the UI and exported as `TFX_PROJECT`. Defaults to the config directory name. |
| `description` | `string` | no | Human-readable description for the default or inherited pipeline. |
| `lanes` | `[]string` | no | Allowed execution lanes. Defaults to `preview`, `canary`, `production`. |
| `default_lane` | `string` | no | Initial selected lane. Falls back to the first item in `lanes`. |
| `require_approval` | `bool` | no | When `true`, the form requires approval before a run can start. |
| `secret_prompt` | `string` | no | Label used in the form for the secret prompt. |
| `secret_env` | `string` | no | Environment variable populated with the collected secret for every step. |
| `working_dir` | `string` | no | Base directory for step execution. Relative values resolve from the config file directory. |
| `before_all` | `[]step` | no | Commands that run before the main pipeline steps. |
| `after_all` | `[]step` | no | Commands that run after the main pipeline succeeds. |
| `on_failure` | `[]step` | no | Commands that run when any prior phase fails. |
| `steps` | `[]step` | yes, if no `flows` | Ordered pipeline steps for the simple single-pipeline mode. |
| `default_flow` | `string` | no | Initial named flow when `flows` is used. |
| `default_profile` | `string` | no | Alias for `default_flow`. |
| `flows` | `[]flow` | yes, if no `steps` | Named pipelines selectable from the `Forms` tab. |
| `profiles` | `[]flow` | no | Alias for `flows`. Do not use both keys at once. |

## Flow Fields

Flows inherit missing values from the top-level config.

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `name` | `string` | yes | Stable name shown in the UI and used by `default_flow`. |
| `description` | `string` | no | Flow-specific description shown in the form snapshot. |
| `lanes` | `[]string` | no | Overrides/inherits the top-level lanes. |
| `default_lane` | `string` | no | Overrides/inherits the top-level default lane. |
| `require_approval` | `bool` | no | Overrides the top-level approval requirement. |
| `secret_prompt` | `string` | no | Overrides/inherits the top-level secret label. |
| `secret_env` | `string` | no | Overrides/inherits the top-level secret env var. |
| `working_dir` | `string` | no | Overrides/inherits the top-level working dir. Relative values resolve from the config file directory. |
| `before_all` | `[]step` | no | Flow-specific hooks appended after top-level `before_all`. |
| `after_all` | `[]step` | no | Flow-specific hooks appended after top-level `after_all`. |
| `on_failure` | `[]step` | no | Flow-specific failure hooks appended after top-level `on_failure`. |
| `steps` | `[]step` | no | Flow-specific steps. If omitted, the flow inherits top-level `steps`. |

## Step Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `id` | `string` | no | Stable step identifier. Defaults to `step-N`. |
| `title` | `string` | no | UI title. Defaults to `id`. |
| `detail` | `string` | no | UI detail line. Defaults to the rendered command. |
| `command` | `[]string` | yes | Executed as argv. No shell interpolation is used. |
| `dir` | `string` | no | Step working directory. Relative values resolve from the effective `working_dir`. |
| `lanes` | `[]string` | no | Allow-list of lanes for this step. Empty means all lanes. |
| `env` | `map[string]string` | no | Extra environment variables for this step. |
| `artifacts` | `[]artifact` | no | Declared files expected from the step. |
| `continue_on_error` | `bool` | no | Marks the step as failed but keeps the pipeline running. |

## Artifact Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `name` | `string` | no | Friendly label shown in summaries. Defaults to the file basename. |
| `path` | `string` | yes | Relative to the step `dir` unless absolute. |
| `description` | `string` | no | Extra text kept in the artifact report. |
| `optional` | `bool` | no | Optional artifacts do not fail the step when missing. |

## Runtime Behavior

- `TFX_PROJECT` and `TFX_LANE` are exported for every command.
- If `secret_env` is configured and the user provides a secret, that env var is exported too.
- Step output is streamed into the `Logs` tab in real time.
- Steps skipped by lane are marked as skipped in the stepper/table UI.
- Steps with `continue_on_error: true` are marked as failed, increment warnings, and do not abort the run.
- Commands are run directly through `exec.CommandContext`, not through `sh -c`.
- If multiple named flows exist, the `Forms` tab asks the user to choose one before lane/approval/secret inputs.
- If only one named flow exists, it is selected automatically.

## Session Persistence

`cmd/tfx` persists lightweight wrapper state on disk:

- selected flow
- selected lane
- project name
- last run timestamp
- recent run history

Secrets remain process-local and are not written to disk.

## Flow Overrides

You can preselect a named flow before the UI renders:

- CLI flag: `tfx --flow release`
- CLI alias: `tfx --profile release`
- Env var: `TFX_FLOW=release`
- Env alias: `TFX_PROFILE=release`
- Lane override: `tfx --lane production` or `TFX_LANE=production`
- Immediate run: `tfx --run` or `TFX_RUN=true`
- JSON output: `tfx --json` or `TFX_JSON=true`
- Quiet summary: `tfx --quiet` or `TFX_QUIET=true`

Precedence is:

1. CLI flags
2. Env vars
3. Config defaults

If the requested flow does not exist, `cmd/tfx` exits with an error.
If the requested lane does not belong to the selected flow, `cmd/tfx` exits with an error.
When `--run` is enabled, `cmd/tfx` treats the run as approved, fills the project from config defaults, reads the secret from the configured `secret_env` when present, and starts the flow immediately.
`--flow`, `--lane`, and `--run` are designed to compose, so direct automation
invocations such as `tfx --flow release --lane production --run` are valid.
`--json` and `--quiet` force non-interactive output even when stdout is a TTY.

## Hooks and Artifacts

The wrapper supports both declared and discovered outputs:

- `before_all`, `after_all`, and `on_failure` are first-class YAML keys.
- Step-level `artifacts` let you declare required or optional files.
- The runner also discovers files written into conventional directories such as
  `dist/`, `coverage/`, `build/`, and `artifacts/`.
- Missing non-optional declared artifacts fail the step.
- Final snapshots and `--json` reports include hook execution and artifact summaries.

## Example

```yaml
project: tfx
description: TFX local automation
lanes: [preview, canary, production]
working_dir: .
default_flow: ci
flows:
  - name: ci
    description: Fast local verification
    before_all:
      - id: prep
        title: Prepare workspace
        command: [go, version]
    steps:
      - id: test
        title: Run tests
        command: [go, test, -coverprofile=coverage.out, ./...]

      - id: build
        title: Build binaries
        command: [go, build, -o, dist/tfx, ./cmd/tfx]

  - name: release
    description: Release candidate pipeline
    default_lane: canary
    require_approval: true
    secret_prompt: Release token
    secret_env: TFX_SECRET
    steps:
      - id: test
        title: Run tests
        command: [go, test, ./...]

      - id: vet
        title: Vet packages
        lanes: [canary, production]
        command: [go, vet, ./...]

      - id: coverage
        title: Check coverage threshold
        lanes: [production]
        command: [make, check-coverage]
        continue_on_error: true

      - id: package
        title: Package release artifacts
        lanes: [production]
        detail: Writes release outputs to dist/
        command: [go, build, -o, dist/tfx, ./cmd/tfx]
        artifacts:
          - name: binary
            path: dist/tfx

    on_failure:
      - id: cleanup
        title: Cleanup release state
        command: [pwd]
```

## Recommended Workflow

1. Copy [examples/tfx.yaml](../examples/tfx.yaml) into your project root.
2. Adjust `working_dir`, lanes, flows, and commands to match the repo.
3. Run `go run ./cmd/tfx`.
4. Fill in runtime inputs in the `Forms` tab and start the selected flow.

## Standalone Tool Dogfooding

`cmd/tfx` is not limited to Go package verification or to TFX itself.
It also works well as a wrapper around standalone CLI repositories, generators,
code transformers, linters, formatters, or any other repo-local workflow.

The reusable pattern is:

1. Build the standalone binary or tool entrypoint inside the repo.
2. Run a low-risk smoke command such as `--help`, `version`, or `doctor`.
3. Execute fixture, snapshot, or workflow scenarios through a repo-local harness.
4. Package the resulting binary, logs, diffs, or snapshots as artifacts.

That keeps TFX focused on orchestration and reporting while the target
repository keeps ownership of the real invocation details.

See [docs/dogfooding-external-tools.md](./dogfooding-external-tools.md) for the
generic guidance and [examples/tfx-external-tool.yaml](../examples/tfx-external-tool.yaml)
for a reusable template. If you are specifically dogfooding Morfx as a
standalone CLI, [docs/dogfooding-morfx-standalone.md](./dogfooding-morfx-standalone.md)
and [examples/tfx-morfx-standalone.yaml](../examples/tfx-morfx-standalone.yaml)
show one concrete instantiation.
