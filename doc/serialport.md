# Resolving the serial port

## Linux

Sometimes there is the need to maintain multiple devices, classic `udev` approaches use the vendor and device IDs plus the serial number to add a symlink with a specific name to `/dev` folder. The serial number might not be set in a way that it is possible to distinguish the different devices.

To workaround this one could use the `/dev/serial/by-path` subfolder and the containing files. 

```sh
ls -al  /dev/serial/by-path/*
lrwxrwxrwx 1 root root 13 Sep 22 09:04 /dev/serial/by-path/pci-0000:00:14.0-usb-0:1.1:1.0-port0 -> ../../ttyUSB0
lrwxrwxrwx 1 root root 13 Sep 22 09:04 /dev/serial/by-path/pci-0000:00:14.0-usb-0:6.1:1.0-port0 -> ../../ttyUSB1
lrwxrwxrwx 1 root root 13 Sep 22 09:04 /dev/serial/by-path/pci-0000:00:14.0-usb-0:7.2:1.0-port0 -> ../../ttyUSB2
lrwxrwxrwx 1 root root 13 Sep 22 09:04 /dev/serial/by-path/pci-0000:00:14.0-usb-0:7.3:1.0 -> ../../ttyACM0
```

So instead of using `/dev/ttyACM0` the more robust approach would be to use `/dev/serial/by-path/pci-0000:00:14.0-usb-0:7.3:1.0`. This will be independent of enumeration orders.

## Keep ModemManager away from the device

Many Linux distributions run [ModemManager](https://modemmanager.org), which opens new USB serial
ports to find out whether a modem is behind them, and sends AT commands such as `AT\r` to them. Norn
does not understand these bytes. Firmware built with norn v0.2.0 or older can get out of step with the
host because of them, and then no longer answers properly until it is restarted
([nornforge/norn#2](https://github.com/nornforge/norn/issues/2)). Newer versions skip such bytes, but the
probing still gets in the way: while ModemManager has the port open, `norn` cannot open it.

Whether ModemManager probes the device depends on the filter policy of your distribution. To make sure it
leaves the device alone, tell it to ignore the device with a udev rule:

```sh
sudo tee /etc/udev/rules.d/99-norn.rules <<'RULE'
# Raspberry Pi Pico running the Norn firmware: not a modem
SUBSYSTEMS=="usb", ATTRS{idVendor}=="2e8a", ATTRS{idProduct}=="000a", ENV{ID_MM_DEVICE_IGNORE}="1"
RULE
sudo udevadm control --reload-rules
sudo udevadm trigger
```

Then unplug the device and plug it in again. To check that the rule applies:

```sh
udevadm info -q property -n /dev/ttyACM0 | grep ID_MM_DEVICE_IGNORE
ID_MM_DEVICE_IGNORE=1
```

`2e8a:000a` is the USB ID of the Raspberry Pi Pico in general, so the rule also applies to other Picos
with a USB serial port. `lsusb` shows the ID of the device if yours differs.

If the host has no modem at all, stopping ModemManager works too:

```sh
sudo systemctl disable --now ModemManager
```

In a dev container, ModemManager runs on the host, not in the container, so the rule belongs on the host.
