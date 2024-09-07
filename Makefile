
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
	install -o root -g wheel -m 0755 scripts/mkinitrd.debian /usr/local/bin/mkinitrd.debian
	install -o root -g wheel -m 0755 scripts/nbctl.py /usr/local/bin/nbctl

