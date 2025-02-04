import json
import discord
import os
import logging
import llm
from pathlib import Path
import db_process
import argparse
import tempfile
from dotenv import load_dotenv
from io import BytesIO  # Use BytesIO instead of StringIO
import asyncio

# -------------------- Private Implementation Details --------------------
def _setup_logging():
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

async def _post_to_channel(channel: discord.TextChannel, db_freindly: dict, message: discord.Message, report_data: dict) -> None:
    position_name = db_freindly.get('position', {}).get('name', '')
    company_name = db_freindly.get('company', '')
    job_url = message.content[23:] or "URL missing!"
    bullets = report_data.get('tailored_bullets', '')
    match_before = report_data.get('initial_score', '')
    match_after = report_data.get('optimized_score', '')
    level = db_freindly.get('position', {}).get('level', {})

    bullets_json = json.dumps(bullets, indent=4)
    file_like = BytesIO(bullets_json.encode('utf-8'))
    file = discord.File(file_like, filename="bullets.json")

    await channel.send(
        f"**Position:** {position_name}\n"
        f"**Level:** {level}\n"
        f"**Company:** {company_name}\n"
        f"**URL:** {job_url}\n"
        f"**Before Babita ji's Tips:** {match_before}\n"
        f"**After Babita ji's Tips:** {match_after}\n\n",
        file=file, 
    )

from typing import Union
def _is_valid_job_message(message: discord.Message, bot_user: Union[discord.User, discord.Member]) -> bool:
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

                    loop = asyncio.get_running_loop()
                    llm_parsed: str = await loop.run_in_executor(None, llm.parse_job_desc, file_path)

                    job_data = json.loads(llm_parsed)
                    db_friendly = job_data.get("db_friendly", {})
                    if not isinstance(db_friendly, dict):
                        raise TypeError("Missing db_friendly data structure")
                    if not db_friendly:
                        await message.reply("Could not parse the job description")
                        return
                    db_process.add_job_to_db(db_friendly, message.id, db_config)
                    report_data = llm.report(db_friendly)

                    processed_channel = discord.utils.get(message.guild.text_channels, name='processed-jobs')
                    if not processed_channel:
                        logger.info("Creating missing processed-jobs channel")
                        processed_channel = await message.guild.create_text_channel('processed-jobs')

                    await _post_to_channel(processed_channel, db_friendly, message, report_data)
                except Exception as e:
                    await message.reply(f"ERROR: {str(e)}\n")
                    logger.error(f"Error processing attachment: {e}")
                    return None

    except discord.Forbidden as e:
        logger.error(f"Missing permissions: {e}")
        await message.reply(f"Permission denied! Error: {str(e)}")
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
            reconnect=True,
            max_retries=10,
            heartbeat_timeout=100
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
                        await process_discord_job(message, self.db_config)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--test", action="store_true", help="Run in test mode")
    args = parser.parse_args()

    config = _get_env_config(args.test)
    bot = JobBot(config['db'])
    bot.run(config['discord_token'])

if __name__ == "__main__":
    main()
