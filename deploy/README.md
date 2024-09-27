# cloudboot

Install netboot.rstms.net 
Run from rigel.rstms.net, this Makefile creates an installer tarball for configuring netboot.rstms.net

## initial install manual commands:
```
doas make cloudboot.tgz
scp cloudboot.tgz cloudboot:.
ssh cloudboot.tgz
cd /
tar zxhf home/mkrueger/cloudboot.tgz
./install.site
```

## update:
```
doas ./update_cloudboot
```

## update mirrors:
```
doas ./update_mirrors
```

## update certs:
```
doas ./update_certs
```
