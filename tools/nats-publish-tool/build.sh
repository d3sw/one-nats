rm -f one-nats-pub.exe
rm -f one-nats-pub
rm -f one-nats-pub-windows.zip 
rm -f one-nats-pub-linux.zip

CGO_ENABLED=0 GOOS=windows go build -a -ldflags '-extldflags "-static"' -o one-nats-pub.exe .
CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o one-nats-pub .

zip one-nats-pub-windows.zip one-nats-pub.exe
zip one-nats-pub-linux.zip one-nats-pub

rm -f one-nats-pub.exe
rm -f one-nats-pub

