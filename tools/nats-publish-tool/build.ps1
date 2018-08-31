go build -o one-nats-pub.exe
docker run -it --rm -v ${pwd}:/app golang bash -c "cd /app && go get -d . && go build -o one-nats-pub"

Compress-Archive -Path one-nats-pub.exe -Update -DestinationPath one-nats-pub-window.zip
Compress-Archive -Path one-nats-pub -Update -DestinationPath one-nats-pub-linux.zip

remove-item one-nats-pub.exe
remove-item one-nats-pub

