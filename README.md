# fleeting-plugin-openstack

GitLab fleeting plugin for OpenStack.

https://docs.gitlab.com/runner/executors/docker_autoscaler.html

## Plugin Configuration

The following parameters are supported:

| Parameter       | Type   | Description                                                                                                         |
| --------------- | ------ | ------------------------------------------------------------------------------------------------------------------- |
| `cloud`         | string | Name of the cloud config from clouds.yaml to use                                                                    |
| `clouds_config` | string | Optional. Path to clouds.yaml                                                                                       |
| `name`          | string | Name of the Auto Scaling Group                                                                                      |
| `boot_time`     | string | Optional. Maximum wait time for instance to boot up. During that time plugin check Cloud-Init signatures.           |
| `server_spec`   | object | Server spec used to create instances. See: [Compute API](https://docs.openstack.org/api-ref/compute/#create-server) |

### Default connector config

| Parameter                | Default |
| ------------------------ | ------- |
| `os`                     | `linux` |
| `protocol`               | `ssh`   |
| `username`               | `unset` |
| `use_static_credentials` | `true`  |

### SSH Key Authentication

The plugin supports two methods for SSH key authentication:

1. **OpenStack Keypairs (Traditional)**: Pre-register SSH keys in OpenStack and reference them via `key_name` in `server_spec`
2. **Cloud-Init User Data**: Embed SSH public keys directly in the instance's user data using cloud-init's `ssh_authorized_keys`

When using SSH keys via user data:

- Set `use_static_credentials = false` in `connector_config`
- Provide `key_path` pointing to your private key file
- Do NOT specify `key_name` in `server_spec`
- Add your public key to `user_data` using cloud-init's `ssh_authorized_keys` directive

See `example_config_userdata_ssh.toml` for a complete example.

## OpenStack setup

1. You should create a special user (recommended) and project (optional),
   then export clouds.yaml with credentials for that cloud.

2. You may create a tenant network for workers, in that case don't forget to add a router.
   In that case manager VM should have two ports: external and that tenant network,
   so it will be able to connect to the worker instances.

3. You should upload a special image with gitlab-runner and container runtime installed in it.
   For example we use [Fedora 38 with Podman](https://mirror.sardinasystems.com/images/Fedora-Cloud-Gitlab-Runner-38-1.6.x86_64.qcow2).

4. **SSH Key Setup (Choose one method)**:
   - **Method A (Cloud-Init)**: Generate an SSH keypair for the manager instance. The public key will be embedded in user data via cloud-init's `ssh_authorized_keys`. No OpenStack keypair registration needed.
   - **Method B (OpenStack Keypairs)**: Generate an SSH keypair and register the public key in Nova/OpenStack, then reference it via `key_name` in the server spec.

## Example runner config

### Example 1: Using OpenStack Keypairs (Traditional)

```
concurrent = 16
check_interval = 0
shutdown_timeout = 0
log_level = "info"

[session_server]
session_timeout = 1800

[[runners]]
name = "manager"
url = "https://gitlab.com"
token = "token"
executor = "docker-autoscaler"
output_limit = 10240
shell = "bash"
environment = [
  "FF_NETWORK_PER_BUILD=1"
  ]

[runners.cache]
Type = "s3"
Shared = true

[runners.cache.s3]
ServerAddress = "s3.foo.bar"
AccessKey = "access"
SecretKey = "secret"
BucketName = "cache"

[runners.docker]
disable_entrypoint_overwrite = false
oom_kill_disable = false
disable_cache = true
shm_size = 0
network_mtu = 0
host = "unix:///run/user/1000/podman/podman.sock"
tls_verify = false
image = "quay.io/podman/stable"
privileged = true
pull_policy = ["always", "always"]

[runners.autoscaler]
capacity_per_instance = 1
max_use_count = 10
max_instances = 16
plugin = "fleeting-plugin-openstack"

[runners.autoscaler.plugin_config]
cloud = "runner"
# clouds_file = "/etc/openstack/clouds.yaml"
name = "podman-runners"
boot_time = "10m"

[runners.autoscaler.plugin_config.server_spec]
name = "podman-runner-%d"                                               # %d replaced with instance index
description = "GitLab CI Podman runners with autoscaling"
tags = ["GitLab", "CI", "Podman"]
imageRef = "d5460af5-83f3-47d7-9c4f-80294c66b267"                       # Fedora 38 + gitlab-runner
flavorRef = "4e9d4fa4-a703-4850-8bc1-58b5e139ab57"                      # xlarge flavor
key_name = "runner_key"                                                 # SSH public key for worker nodes
networks = [ { uuid = "f05e7f64-9e0f-4c5c-acb0-b636000d7301" } ]        # tenant network
security_groups = [ "cee22d91-bb9a-455d-be88-e911d3cb066a" ]            # allow SSH ingress from tenant network
scheduler_hints = { group = "a9c941cb-5b34-46e0-8fc6-7471e3b77c75" }    # [Soft-]Anti-Affinity group
# runcmd required for podman, see also gitlab docs
user_data = '''#cloud-config
package_update: true
package_upgrade: true
runcmd:
- systemctl --now enable podman.socket
- sudo -u fedora systemctl --user --now enable podman.socket
- loginctl enable-linger gitlab-runner
'''

[runners.autoscaler.connector_config]
username = "fedora"
password = ""
key_path = "/etc/gitlab-runner/id_rsa"
use_static_credentials = true
keepalive = "30s"
timeout = "0m"
use_external_addr = false

[[runners.autoscaler.policy]]
idle_count = 2
idle_time = "30m0s"
scale_factor = 0.0
scale_factor_limit = 0
```

### Example 2: Using SSH Keys via Cloud-Init User Data

See [`example_config_userdata_ssh.toml`](./example_config_userdata_ssh.toml) for a complete example that embeds SSH public keys in user data without requiring OpenStack keypair registration.

Key differences:

- Set `use_static_credentials = false`
- Omit `key_name` from `server_spec`
- Add `ssh_authorized_keys` to `user_data`
