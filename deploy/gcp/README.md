# Google cloud deployment

The game runs on a single pay-as-you-go `e2-micro` VM in project `el-poblador`,
zone `southamerica-west1-a`, with a 10 GB standard persistent boot disk. Its
reserved Standard Tier IPv4 address is `34.0.59.165` (`el-poblador-ip`).

The custom VPC `el-poblador` contains subnet `el-poblador-santiago`
(`10.42.0.0/24`). Firewall rule `el-poblador-game` allows public TCP port 2222;
`el-poblador-admin-iap` allows TCP port 22 only from the IAP range
`35.235.240.0/20`. Both target VMs tagged `el-poblador`. The VM has no service
account. The game runs as an unprivileged system user under systemd and starts
automatically at boot.

Connect with:

```sh
ssh -p 2222 yourname@el-poblador.vichoreyes.cl
```

## Administration

Administrative SSH goes through GCP IAP. An administrator needs the appropriate
Compute Engine and IAP permissions in the project.

```sh
gcloud compute ssh el-poblador --project=el-poblador \
  --zone=southamerica-west1-a --tunnel-through-iap
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
  --project=el-poblador --zone=southamerica-west1-a --tunnel-through-iap
gcloud compute ssh el-poblador --project=el-poblador \
  --zone=southamerica-west1-a --tunnel-through-iap \
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
  --zone=southamerica-west1-a --delete-disks=all
gcloud compute addresses delete el-poblador-ip --project=el-poblador \
  --region=southamerica-west1
gcloud compute firewall-rules delete el-poblador-game el-poblador-admin-iap \
  --project=el-poblador
gcloud compute networks subnets delete el-poblador-santiago \
  --project=el-poblador --region=southamerica-west1
gcloud compute networks delete el-poblador --project=el-poblador
```

Remove the Cloudflare `el-poblador` DNS record as well. These commands leave the
project and its pre-existing default network intact.
