docker build -t discord-bot .
docker run -d --network="host" discord-bot:latest
