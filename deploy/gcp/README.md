# Google cloud deployment

The game runs on a single non-preemptible `e2-micro` VM in project `el-poblador`,
zone `us-west1-b` (Oregon), with a 10 GB standard persistent boot disk. Its
reserved Standard Tier IPv4 address is `35.212.215.46` (`el-poblador-ip`).

The custom VPC `el-poblador` contains subnet `el-poblador-oregon`
(`10.43.0.0/24`). Firewall rule `el-poblador-game` allows public TCP port 2222;
`el-poblador-admin-iap` allows TCP port 22 only from the IAP range
`35.235.240.0/20`. Both target VMs tagged `el-poblador`. The VM has no service
account. The game runs as an unprivileged system user under systemd and starts
automatically at boot.

Connect with:

```sh
ssh -p 2222 yourname@catan.vichoreyes.cl
```

## Costs

Oregon is eligible for the [Compute Engine Free Tier](https://cloud.google.com/free/docs/free-cloud-features#compute).
The allowance covers one month's worth of non-preemptible `e2-micro` usage and
30 GB-months of standard persistent disk, shared across eligible regions and
projects on the billing account. This deployment uses one VM and a 10 GB disk;
other usage on the same billing account can consume those allowances.

The deployment is not entirely free: the public IPv4 address costs
[$0.005/hour](https://cloud.google.com/vpc/network-pricing#ipaddress)
(about US$3.65 for 730 hours). Network traffic beyond the applicable free
allowances is also billed. The DNS record is DNS-only because players connect
directly over SSH on port 2222.

## Administration

Administrative SSH goes through GCP IAP. An administrator needs the appropriate
Compute Engine and IAP permissions in the project.

```sh
gcloud compute ssh el-poblador --project=el-poblador \
  --zone=us-west1-b --tunnel-through-iap
```

On the VM:

```sh
sudo systemctl status el-poblador
sudo journalctl -u el-poblador -n 100 --no-pager
sudo systemctl restart el-poblador
```

The server's SSH host key lives in
`/var/lib/el-poblador/ssh_host_ed25519_key`. Preserve it across updates to avoid
host-key warnings for players. Multiplayer lobbies are held in memory; restarting
the process or VM interrupts active games. Rejoining an active game is currently
unsupported.

## Updating the binary

From the repository root, with Go and gcloud installed:

```sh
go test ./...
payload_dir=$(mktemp -d)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$payload_dir/el-poblador" .
cp deploy/gcp/el-poblador.service deploy/gcp/install.sh "$payload_dir/"
gcloud compute scp --recurse "$payload_dir" el-poblador:~/el-poblador-update \
  --project=el-poblador --zone=us-west1-b --tunnel-through-iap
gcloud compute ssh el-poblador --project=el-poblador \
  --zone=us-west1-b --tunnel-through-iap \
  --command='sudo sh ~/el-poblador-update/install.sh ~/el-poblador-update && rm -rf ~/el-poblador-update'
rm -r "$payload_dir"
```

Finish any games before updating: installation restarts the service. The install
script preserves the state directory and host key. Remove any leftover remote
`~/el-poblador-update` directory before retrying a failed upload.

## Removing the deployment

Stopping the VM stops compute charges, but retained disks and the reserved IPv4
address continue to incur charges. To remove this deployment entirely, first
save anything you need from the VM, then run the following destructive commands:

```sh
gcloud compute instances delete el-poblador --project=el-poblador \
  --zone=us-west1-b --delete-disks=all
gcloud compute addresses delete el-poblador-ip --project=el-poblador \
  --region=us-west1
gcloud compute firewall-rules delete el-poblador-game el-poblador-admin-iap \
  --project=el-poblador
gcloud compute networks subnets delete el-poblador-oregon \
  --project=el-poblador --region=us-west1
gcloud compute networks delete el-poblador --project=el-poblador
```

Remove the Cloudflare `catan` DNS record as well. These commands leave the
project and its pre-existing default network intact.
