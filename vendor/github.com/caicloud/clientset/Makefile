all: gen

clean:
	rm -rf informers kubernetes listers
	find ./pkg/apis -maxdepth 3 -mindepth 3 -name 'zz_generated.*.go' -exec rm -f {} \;

gen: clean
	cp -r expansions/* ./
	bash cmd/autogenerate.sh

