maelstrom_deps:
	sudo apt-get update
	sudo apt-get install graphviz gnuplot

maelstrom_fetch:
	wget -P ./third-party https://github.com/jepsen-io/maelstrom/releases/download/v0.2.3/maelstrom.tar.bz2 && bzip2 -d ./third-party/maelstrom.tar.bz2 && tar -xvf ./third-party/maelstrom.tar -C ./third-party && rm ./third-party/maelstrom.tar

maelstrom_serve:
	./third-party/maelstrom/maelstrom serve

test_g-set:
	go build -o ./bin/g-set ./g-set && ./third-party/maelstrom/maelstrom test -w g-set --bin ./bin/g-set --time-limit 30 --rate 10 --nemesis partition

test_g-counter:
	go build -o ./bin/counter ./counter && ./third-party/maelstrom/maelstrom test -w g-counter --bin ./bin/counter --time-limit 20 --rate 10

test_pn-counter:
	go build -o ./bin/counter ./counter && ./third-party/maelstrom/maelstrom test -w pn-counter --bin ./bin/counter --time-limit 30 --rate 10 --nemesis partition