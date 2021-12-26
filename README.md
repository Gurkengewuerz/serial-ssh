# Serial SSH

Serial ssh is a cross-platform serial proxy.
The main idea was to connect a tiny Raspberry Pi Zero W or alternatives with WiFi to a [Blade Server](https://twitter.com/Gurkengewuerz/status/1472999872014036996) to get the onboard serial interface available via ssh.

### Docker
Docker is only support on linux. On Windows it is very difficult to pass COM ports into the container.
```bash
docker run --rm -it \
  -p 2222:2222 \
  -e COM_PORT=/dev/ttyUSB0 \
  -v /dev/ttyUSB0:/dev/ttyUSB0 \
  -v $(pwd)/banner:/banner \
  -v $(pwd)/sshkeys:/sshkeys \
  gurkengewuerz/serial-ssh:latest
```