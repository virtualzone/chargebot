#!/bin/sh
VERSION=0.2.3
rm -f /tmp/v${VERSION}.tar.gz
wget -P /tmp/ https://github.com/teslamotors/fleet-telemetry/archive/refs/tags/v${VERSION}.tar.gz
tar xzf /tmp/v${VERSION}.tar.gz -C /tmp
cp /tmp/fleet-telemetry-${VERSION}/protos/vehicle_data.* .
rm -rf /tmp/fleet-telemetry-${VERSION}
rm -f /tmp/v${VERSION}.tar.gz