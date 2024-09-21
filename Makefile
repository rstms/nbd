
bin = nbd

$(bin): fmt 
	fix go build

run:
	./$(bin)

fmt:
	fix go fmt ./...

clean:
	go clean


install:
	install -o root -g wheel -m 0755 ./$(bin) /usr/local/bin/$(bin)
	install -o root -g wheel -m 0755 scripts/rc /etc/rc.d/$(bin)
	install -o root -g wheel -m 0700 scripts/mkboot.debian /root/mkboot.debian
	install -o root -g wheel -m 0700 scripts/mkboot.openbsd /root/mkboot.openbsd
	install -o root -g wheel -m 0700 scripts/nbdperm /root/nbdperm
	install -o root -g netboot -m 0750 scripts/nbctl.py /usr/local/bin/nbctl
	rcctl restart nbd

