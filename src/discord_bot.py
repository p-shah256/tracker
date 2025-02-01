import discord
import os
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

# Get the bot token from the environment
TOKEN = os.getenv('DISCORD_BOT_TOKEN')
CHANNEL_ID = os.getenv('CHANNEL_ID')

if not TOKEN:
    raise ValueError("No DISCORD_BOT_TOKEN found in .env file.")

# Intents are required to access message content
intents = discord.Intents.default()
intents.message_content = True  # Enable access to message content

# Create a client instance
client = discord.Client(intents=intents)

@client.event
async def on_ready():
    print(f'Logged in as {client.user}')

    # Debug: List all guilds (servers) the bot is in
    print(f"Bot is in {len(client.guilds)} guild(s):")
    for guild in client.guilds:
        print(f"- {guild.name} (ID: {guild.id})")

    # Find the first text channel the bot can access
    channel = None
    for guild in client.guilds:
        print(f"Checking channels in {guild.name}:")
        for ch in guild.text_channels:
            print(f"- #{ch.name} (ID: {ch.id})")
            if ch.permissions_for(guild.me).send_messages:  # Check if the bot can send messages
                channel = ch
                print(f"Found suitable channel: #{ch.name} (ID: {ch.id})")
                break
        if channel:
            break

    if channel:
        # Send a welcome message
        await channel.send("Bot is online! Let's have some fun!")
        print(f"Bot sent a welcome message to #{channel.name}.")

        # Fetch and print message history
        print(f"Fetching message history from #{channel.name}:")
        async for message in channel.history(limit=100):  # Adjust limit as needed
            print(f'{message.author}: {message.content}')
    else:
        print("No suitable channel found.")

@client.event
async def on_message(message):
    # Log all new messages
    print(f'New message: {message.author}: {message.content}')

# Run the bot
client.run(TOKEN)
