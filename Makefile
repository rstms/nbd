
bin = netboootd
swagger_version = 5.17.14

$(bin): fmt swagger
	fix go build

fmt:
	fix go fmt

clean:
	go clean
	rm -rf swagger

sterile: clean
	rm -f *.tar.gz

install: $(bin)
	install -o root -g wheel -m 0755 ./$(bin) /usr/local/bin/$(bin)
	install -o root -g wheel -m 0755 ./rc /etc/rc.d/$(bin)

v$(swagger_version).tar.gz:
	curl -LO swagger.tgz https://github.com/swagger-api/swagger-ui/archive/refs/tags/v$(swagger_version).tar.gz

swagger: v$(swagger_version).tar.gz
	tar -zxf v$(swagger_version).tar.gz
	rm -rf swagger
	mv swagger-ui-$(swagger_version)/dist swagger
	rm -rf swagger-ui-$(swagger_version)
	sed -e 's/https:\/\/petstore.swagger.io/http:\/\/rigel:8080/' -i swagger/swagger-initializer.js

