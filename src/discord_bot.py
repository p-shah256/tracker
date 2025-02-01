import discord
import os
import logging
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

# Set up logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Get the bot token from the environment
TOKEN = os.getenv('DISCORD_BOT_TOKEN')

if not TOKEN:
    logger.error("No DISCORD_BOT_TOKEN found in .env file.")
    raise ValueError("No DISCORD_BOT_TOKEN found in .env file.")

# Intents are required to access message content
intents = discord.Intents.default()
intents.message_content = True  # Enable access to message content

# Create a client instance
client = discord.Client(intents=intents)

@client.event
async def on_ready():
    logger.info(f'Logged in as {client.user}')

    logger.debug(f"Bot is in {len(client.guilds)} guild(s):")
    for guild in client.guilds:
        logger.debug(f"- {guild.name} (ID: {guild.id})")

    channel = next(
        (ch for guild in client.guilds for ch in guild.text_channels 
         if ch.permissions_for(guild.me).send_messages),
        None
    )

    if channel:
        await channel.send("Bot is online! Let's have some fun!")
        logger.info(f"Bot sent a welcome message to #{channel.name}.")

        logger.debug(f"Fetching message history from #{channel.name}:")
        async for message in channel.history(limit=100):  # Adjust limit as needed
            if client.user in message.mentions:
                logger.info(f'Bot mentioned = {message.author}: {message.content}')
    else:
        logger.warning("No suitable channel found.")


@client.event
async def on_message(message):
    if message.author == client.user:
        return
    if client.user not in message.mentions:
        return
    if not message.attachments:
        await message.reply(f"404 HTML Not Found! {message.author.mention}, did your file take a coffee break? ☕")
        return

    html_files = [att for att in message.attachments if att.filename.endswith('.html')]
    if not html_files:
        await message.reply(f"Plot twist, {message.author.mention}! That's as much HTML as a potato. 🥔 Need a .html file!")
        return

    await message.reply(f"Jackpot, {message.author.mention}! HTML spotted in the wild! 🎯 Time to parse this bad boy... 🕵️")
    for html_file in html_files:
        logger.info(f"HTML file received: {html_file.filename}")
        # await html_file.save(html_file.filename)

# Run the bot
client.run(TOKEN)
