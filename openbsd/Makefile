major = 7
minor = 5
version = $(major).$(minor)
label = $(major)$(minor)
arch = amd64

src_file = cd$(label).iso
dst_file = netboot$(label).iso

src = /var/www/htdocs/pub/OpenBSD/$(version)/$(arch)/$(src_file)
dst = /var/www/netboot/$(dst_file)


$(dst): $(src) Makefile mkcd rc.netboot certs
	doas ./mkcd $(src) $(dst_file)
	doas mv $(dst_file) $(dst)
	rm -rf certs

certs: 
	mkdir certs
	cp /etc/ssl/netboot.pem certs
	cp /etc/ssl/netboot.key certs

clean:
	rm -rf certs
