import discord
import os
import logging
from cleaner import clean_html
import llm
from pathlib import Path
from dotenv import load_dotenv

import db_process

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


intents = discord.Intents.default()
intents.message_content = True  # Enable access to message content
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
        logger.debug(f"Fetching message history from #{channel.name}: and looking for UNPROCESSED messages")
        async for message in channel.history(limit=100):
            await on_message(message)

    else:
        logger.warning("No suitable channel found.")


@client.event
async def on_message(message: discord.Message):
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

    for html_file in html_files:
        logger.info(f"HTML file received: {html_file.filename}")
        file_path = Path(f"/tmp/{html_file.filename}")
        await html_file.save(file_path)
        await process_valid(message, file_path)
        os.remove(file_path)
        logger.info(f"File removed: {file_path}")

async def process_valid(message: discord.Message, file_path: Path):
    if db_process.if_processed(message.id, db_config):
        return

    await message.reply(f"Jackpot, {message.author.mention}! HTML spotted in the wild! 🎯 Time to parse this bad boy... 🕵️")
    logger.info(f"Unprocessed Message found {message.author}: {message.content}")
    llm_response = llm.get_llm_response(str(file_path))
    db_freindly = db_process.process_job_posting(llm_response, message.id, db_config)
    await message.reply(
        f"🚀 While your competition is still reading the requirements, I've data-mined this bad boy into " 
        f"{len(db_freindly['skills'])} skills and requirements! Want the full specs? Just hit me with a !details"
    )


import argparse

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--test", action="store_true", help="Run in test mode")
    args = parser.parse_args()

    load_dotenv()

    required_vars = ['DISCORD_BOT_TOKEN', 'DB_PASS', 'DB_USER', 'DB_PORT', 'DB_HOST']
    env_vars = {var: os.getenv(var) for var in required_vars}

    db_config = {
        "dbname": "postgres",
        "user": env_vars['DB_USER'],
        "password": env_vars['DB_PASS'],
        "host": "localhost" if not args.test else env_vars['DB_HOST'],  # Use localhost in test mode
        "port": env_vars['DB_PORT'],
    }

    # Print the database configuration for debugging
    print(db_config)

    client.run(env_vars['DISCORD_BOT_TOKEN'])
