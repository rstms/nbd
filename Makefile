
bin != basename $$(pwd)

latest_release != gh release list --json tagName --jq '.[0].tagName' | tr -d v
version != cat VERSION

gitclean = if git status --porcelain | grep '^.*$$'; then echo git status is dirty; false; else echo git status is clean; true; fi


$(bin): fmt 
	fix go build

run:
	./$(bin)

fmt:
	fix go fmt ./...

release:
	@$(gitclean) || { [ -n "$(dirty)" ] && echo "allowing dirty release"; }
	@$(if $(update),gh release delete -y v$(version),)
	gh release create v$(version) --notes "v$(version)"

clean:
	rm -f $(program)
	go clean

sterile: clean
	which $(program) && go clean -i || true
	go clean -r || true
	go clean -cache
	go clean -modcache
	rm -f go.mod go.sum

install:
	install -o root -g wheel -m 0755 ./$(bin) /usr/local/bin/$(bin)
	install -o root -g wheel -m 0755 scripts/rc /etc/rc.d/$(bin)
	install -o root -g wheel -m 0700 scripts/mkboot.debian /root/mkboot.debian
	install -o root -g wheel -m 0700 scripts/mkboot.openbsd /root/mkboot.openbsd
	install -o root -g wheel -m 0700 scripts/mkboot.alpine /root/mkboot.alpine
	install -o root -g wheel -m 0700 scripts/nbdperm /root/nbdperm
	install -o root -g wheel -m 0700 scripts/update_mirrors /root/update_mirrors
	install -o root -g netboot -m 0750 scripts/nbctl.py /usr/local/bin/nbctl
	rcctl restart $(bin)


test:
	scripts/mkboot.openbsd 00:0c:29:46:8a:61 com0


.PHONY: deploy
deploy:
	cd deploy; ./update_cloudboot
