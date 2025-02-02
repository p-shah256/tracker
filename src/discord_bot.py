import discord
import os
import logging
import llm
from pathlib import Path
import db_process
import argparse
import tempfile
from dotenv import load_dotenv

# -------------------- Private Implementation Details --------------------
def _setup_logging():
    """Hidden implementation detail for logging configuration"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    return logging.getLogger(__name__)

logger = _setup_logging()

def _get_env_config(is_test: bool = False) -> dict:
    load_dotenv()
    required_vars = {'DB_USER', 'DB_PASS', 'DB_HOST', 'DB_PORT', 'DISCORD_BOT_TOKEN'}
    env_vars = {var: os.getenv(var) for var in required_vars}

    config= {
        "db": {
            "dbname": "postgres",
            "user": env_vars['DB_USER'],
            "password": env_vars['DB_PASS'],
            "host": env_vars['DB_HOST'] if is_test else "localhost",
            "port": env_vars['DB_PORT']
        },
        "discord_token": env_vars['DISCORD_BOT_TOKEN']
    }
    print(config)
    return config

async def _post_to_channel(channel: discord.TextChannel, db_freindly: dict, message: discord.Message) -> None:
    """Jethalal's premium job posting formatter - ekdum first class!"""
    position_name = db_freindly.get('position', {}).get('name', 'Babuchak position not found!')
    company_name = db_freindly.get('company', 'Gada Electronics level company not found!')
    job_url = message.content[23:] or "URL missing! Bilkul Mehta Sahab ki savings account jaisa empty!"
    bullets = db_freindly.get('bullets', 'Iyer ke jokes jaise kuch bhi nahi mila!')
    match_before = db_freindly.get('match_before', 'Popatlal ki shaadi jaise - N/A')
    match_after = db_freindly.get('match_after', 'Jethalal ki diet plan jaise - non-existent!')
    level = db_freindly.get('position', {}).get('level', {})
    await channel.send(
        f"**Position:** {position_name}\n"
        f"**Level:** {level}\n"
        f"**Company:** {company_name}\n"
        f"**URL:** {job_url}\n"
        f"**Babita ji's Tips:** {bullets}\n"
        f"**Before Babita ji's Tips:** {match_before}\n"
        f"**After Babita ji's Tips:** {match_after}\n\n"
    )

from typing import Union
def _is_valid_job_message(message: discord.Message, bot_user: Union[discord.User, discord.Member]) -> bool:
    """Hidden implementation detail for message validation"""
    return all([
        message.author != bot_user,
        bot_user in message.mentions,
        any(att.filename.endswith('.html') for att in message.attachments)
    ])

# -------------------- The One True Public Interface --------------------
async def process_discord_job(message: discord.Message, db_config: dict) -> None:
    if message.guild is None:
        logger.warning("Message is not in a guild, skipping")
        return
    if not _is_valid_job_message(message, message.guild.me):
        return
    if db_process.if_processed(message.id, db_config):
        logger.info(f"Message {message.id} already processed, skipping")
        return
    await message.add_reaction("⏳")

    try:
        for attachment in message.attachments:
            if not attachment.filename.endswith('.html'):
                    await message.reply("Arey babuchak! HTML files only! Ye kya non-HTML file bhej diya? Bilkul Bagha jaise kaam karta hai tu!")
                    return 

            with tempfile.NamedTemporaryFile(suffix=".html", delete=True) as temp_file:
                file_path = Path(temp_file.name)
                try:
                    await attachment.save(file_path)
                    llm_data = llm.parse(file_path)
                    db_freindly = db_process.process_job_posting(llm_data, message.id, db_config)
                    processed_channel = discord.utils.get(message.guild.text_channels, name='processed-jobs')
                    if not processed_channel:
                        logger.info("Creating missing processed-jobs channel")
                        processed_channel = await message.guild.create_text_channel('processed-jobs')
                    if not db_freindly:
                        await message.reply("Hey Bhagwan! LLM ne jawab hi nahi diya! Bilkul Iyer jaise chup ho gaya!")
                        return
                    report = llm.report(db_freindly)
                    await _post_to_channel(processed_channel, db_freindly, message)
                except Exception as e:
                    await message.reply(f"Arey bapre bap! Processing mein gadbad ho gayi: {str(e)}\nBilkul Mehta Sahab ke calculations jaise!")
                    logger.error(f"Error processing attachment: {e}")
                    return None

    except discord.Forbidden as e:
        logger.error(f"Missing permissions: {e}")
        await message.reply(f"Aye halo! Permission denied! Bilkul Sodhi ke shop jaise locked hai sab! Error: {str(e)}")
    except discord.HTTPException as e:
        logger.error(f"HTTP error: {e}")
        await message.reply(f"Babuchak network! Champaklal ki walking speed se bhi slow hai! Error: {str(e)}")
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        await message.reply(f"Arey oh Sampla! Unexpected error aa gaya! Bilkul Daya ben ki rasoi jaise unpredictable! Error: {str(e)}")
    finally:
        await message.clear_reactions()
        await message.add_reaction("✅")

# -------------------- Bot Implementation --------------------
class JobBot(discord.Client):
    def __init__(self, db_config: dict):
        intents = discord.Intents.default()
        intents.message_content = True
        super().__init__(
            intents=intents,
            reconnect=True,  # Ensure automatic reconnection
            max_retries=10,  # Maximum reconnection attempts
            heartbeat_timeout=100  # Longer heartbeat timeout
        )
        self.db_config = db_config
        self.db_config = db_config

    async def on_message(self, message: discord.Message) -> None:
        logger.info("NEW message received")
        await process_discord_job(message, self.db_config)

    async def on_ready(self):
        logger.info(f'Bot is ready and connected to {len(self.guilds)} guild(s)')
        for guild in self.guilds:
            for channel in guild.text_channels:
                if channel.permissions_for(guild.me).send_messages:
                    async for message in channel.history(limit=20):
                        logger.info("Starting to process older messages")
                        await process_discord_job(message, self.db_config)


def main():
    """Entry point with minimal complexity"""
    parser = argparse.ArgumentParser()
    parser.add_argument("--test", action="store_true", help="Run in test mode")
    args = parser.parse_args()

    config = _get_env_config(args.test)
    bot = JobBot(config['db'])
    bot.run(config['discord_token'])

if __name__ == "__main__":
    main()
