# pjlink_exporter
PJLink Exporter for Prometheus

This exporter uses the pjlink specifications according to https://pjlink.jbmia.or.jp/english/data_cl2/PJLink_5-1.pdf

However, only commands according to PJLink Class 1 are implemented yet.

## Usage

```sh
./pjlink_exporter
```

Visit http://localhost:2112/pjlink?target=mybeamername.localnetwork where mybeamername.localnetwork is the IP or DNS-Name of the
PJLink device to get metrics from.

## Installation

Clone this repository from github to your go directory. Within this repository run:

```
make build
```

Copy build to your /usr/local/bin

```
cp pjlink_exporter /usr/local/bin
```

Create config file

```
mkdir /etc/pjlink_exporter/
vim /etc/pjlink_exporter/pjlink.yml
```

Edit config file as described in section Configuration

Change permissions
```
chown prometheus:prometheus /etc/pjlink_exporter/pjlink.yml
chown prometheus:prometheus /usr/local/bin/pjlink_exporter
```

Setup exporter as daemon

```
vim /etc/init.d/pjlink_exporter
chmod +x /etc/init.d/pjlink_exporter
update-rc.d pjlink_exporter defaults

vim /etc/default/pjlink_exporter
START=yes

```


## Configuration

The pjlink exporter reads from a `pjlink.yml` config file by default.

Within the `pjlink.yml` config file, specify the default pjlink password. This password will be used for all devices as default.

If any device has a special password, exceptions can be specified for each host. Hosts are specified by IP-Adress or DNS-Name.

```YAML
## password specifies the default password for all devices
password: defPass

## in the devices section, specify password exceptions for any hosts. If host is not specified, the default password rules
devices:
  - host: my-fancy-beamer1.localnetwork
    pass: canaryPass
  - host: my-fancy-beamer2.localnetwork
    pass: anotherCanaryPass
```

## Prometheus Configuration

The pjlink exporter needs to be passed the address as a parameter, this can be
done with relabelling.

Example config:
```YAML
scrape_configs:
  - job_name: 'pjlink'
    static_configs:
      - targets:
        - my-fancy-beamer1.localnetwork  # PJLink device.
    metrics_path: /pjlink
    params:
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: 127.0.0.1:2112  # The PJLink exporter's real hostname:port.
```

## Contributions
Thanks to https://github.com/prometheus/snmp_exporter. This project was used as example and for inspriration while realizing this exporter.