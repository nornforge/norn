# Systemd Service


# Installation of the Systemd service file

Please copy the file provided in [scripts/systemd/norn.service](scripts/systemd/norn.service) to `/etc/systemd/system/norn.service` by using the `sudo` command.

```sh
sudo cp scripts/systemd/norn.service /etc/systemd/system/norn.service
```


## Serial port definition

As mentioned in the [serial port documentation](doc/serialport.md) it is recommended to use the serial by path approach to clearly identify the correct serial port. To allow the systemd service file to be independent of the serial configuration the recommended way is to use environment variables.


```sh
sudo systemctl edit norn
```

If you prefer like myself the usage of vim for such kind of work you can use

```sh
sudo EDITOR=vim systemctl edit norn
```


This uses the system editor to manipulate the file. Use the serial port you have figured out based on the information from the [serial port documentation](doc/serialport.md).

```sh
ls /dev/serial/by-path/pci-0000\:00\:14.0-usb-0\:7.1\:1.0 -al
lrwxrwxrwx 1 root root 13 Aug 13 14:37 /dev/serial/by-path/pci-0000:00:14.0-usb-0:7.1:1.0 -> ../../ttyACM0
```

In this example it is `/dev/serial/by-path/pci-0000:00:14.0-usb-0:7.1:1.0`

```ini
[Service]
Environment="NORN_SERIAL_PORT=/dev/serial/by-path/pci-0000:00:14.0-usb-0:7.1:1.0"  
```

## Troubleshooting

The example systemd service file was tested on Ubuntu, you maybe have to tweak the group for the serial port access from `dialout` to one which fits your distribution.