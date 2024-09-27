# netboot

netboot server management API daemon

### dependencies:
- https://github.com/rstms/gdl
- https://github.com/rstms/utcd
- https://github.com/rstms/ipxe
- rstms keymaster PKI infrastructure

### friends:
- https://github.com/rstms/mkcert
- https://github.com/rstms/boxen
- step CA
- rstms keymaster PKI


### details:
netboot server runs nginx frontend relaying to to utcd and nbd
netboot.rstms.net is a Hetzner VPC
localboot.rstms.net is an alias on rigel.rstms.net
both have NGINX configuration

### root crontab script on rigel:
```
0	*/6	*	*	*	/root/update_rigel_certs >>/var/log/acme-client 2>&1
1	*/6	*	*	*	/root/update_netboot_certs >>/var/log/acme-client 2>&1
```

### SSL Certs
client certs for the IPXE boot images and for boxen are generated
using the mkcert tool and signed by keymaster.rstms.net

netboot server SSL certs are generated on rigel.rstms.net net using
keymaster.rstms.net

see: rigel.rstms.net:/etc/acme-client.conf

`update_netboot_certs`
Uses ipctl to manage local split-horizon DNS, temporarily resolving
netboot.rstms.net to localboot.rstms.net for the acme transaction


### doas.conf on machine running nbd
```
permit keepenv nopass _nbd cmd /root/mkboot.debian
permit keepenv nopass _nbd cmd /root/mkboot.openbsd
```
