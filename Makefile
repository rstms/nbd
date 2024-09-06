
bin = netbootd

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
	install -o root -g wheel -m 0755 ./rc /etc/rc.d/$(bin)

