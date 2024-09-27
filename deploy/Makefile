# install netboot server on cloudboot host

help:
	cat README.md

pkg:
	mkdir -p pkg/var/www/netboot
	mkdir -p pkg/var/www/htdocs/pub
	mkdir pkg/var/www/htdocs/debian
	mkdir -p pkg/etc/nginx
	mkdir -p pkg/etc/ssl/private
	mkdir pkg/etc/rc.d
	mkdir -p pkg/usr/local/bin
	mkdir pkg/root
	chmod 0700 pkg/root
	cp /etc/nginx/nginx.conf pkg/etc/nginx
	awk </etc/nginx/nginx.conf 'BEGIN {count=0} /.*server {.*/{ count += 1} {if (count <1 || count > 3 ) {print $$0}}' >pkg/etc/nginx/nginx.conf
	sed -e 's/192.168.66.6://' -i pkg/etc/nginx/nginx.conf
	sed -e '/server_name/s/ 192.168.66.6//' -i pkg/etc/nginx/nginx.conf
	cp /etc/nginx/require_client_cert.conf pkg/etc/nginx
	cp /var/www/netboot/gdl pkg/var/www/netboot
	cp /var/www/netboot/gdl.tgz pkg/var/www/netboot
	cp /var/www/netboot/netboot* pkg/var/www/netboot
	cp /usr/local/bin/nbd pkg/usr/local/bin/nbd
	cp /usr/local/bin/utcd pkg/usr/local/bin/utcd
	cp /usr/local/bin/nbctl pkg/usr/local/bin/nbctl
	cp /etc/rc.d/nbd pkg/etc/rc.d
	cp /etc/rc.d/utcd pkg/etc/rc.d
	cp /root/mkboot.openbsd pkg/root
	cp /root/mkboot.debian pkg/root
	cp /root/nbdperm pkg/root
	cp /etc/ssl/keymaster.pem pkg/etc/ssl
	cp /etc/ssl/netboot.rstms.net.fullchain.pem pkg/etc/ssl
	cp /etc/ssl/private/netboot.rstms.net.key pkg/etc/ssl/private
	cp -rp remote/* pkg
	ln -s ../htdocs/pub pkg/var/www/netboot/pub
	ln -s ../htdocs/debian pkg/var/www/netboot/debian
	chown -R root.wheel pkg
	chown -R root.www pkg/var/www

cloudboot.tgz: clean pkg
	cd pkg; tar zcfv ../$@ *
	chown mkrueger.mkrueger $@
	rm -rf pkg

certs:
	mkdir -p certs/etc/ssl/private
	cp /etc/ssl/netboot.rstms.net.fullchain.pem certs/etc/ssl
	cp /etc/ssl/private/netboot.rstms.net.key certs/etc/ssl/private


certs.tgz: clean certs
	cd certs; tar zcf ../$@ etc
	chown mkrueger.mkrueger $@
	rm -rf certs

clean:
	rm -rf pkg
	rm -rf certs
	rm -rf certs.tgz
	rm -rf cloudboot.tgz
