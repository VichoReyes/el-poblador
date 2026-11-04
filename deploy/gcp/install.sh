#!/bin/sh
# Run as root with the binary and service file in the supplied directory.
set -eu

payload_dir=${1:?Usage: install.sh PAYLOAD_DIRECTORY}
if ! id el-poblador >/dev/null 2>&1; then
    useradd --system --user-group --home-dir /var/lib/el-poblador \
        --shell /usr/sbin/nologin el-poblador
fi

# Rename the binary atomically so an existing process can keep running until restart.
install -m 0755 "$payload_dir/el-poblador" /usr/local/bin/el-poblador.new
mv /usr/local/bin/el-poblador.new /usr/local/bin/el-poblador
install -m 0644 "$payload_dir/el-poblador.service" /etc/systemd/system/el-poblador.service
systemctl daemon-reload
systemctl enable el-poblador.service
systemctl restart el-poblador.service
systemctl is-active el-poblador.service
