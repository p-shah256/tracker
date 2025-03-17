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
  


from typing import Union
"""
Checks if the message is valid(from bot, mentions bot, has HTML attachment)
"""
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
                    await message.reply("Only HTML files are allowed")
                    return 

            with tempfile.NamedTemporaryFile(suffix=".html", delete=True) as temp_file:
                file_path = Path(temp_file.name)
                try:
                    await attachment.save(file_path)

                    loop = asyncio.get_running_loop()
                    llm_parsed: str = await loop.run_in_executor(None, llm.parse_job_desc, file_path)

                    parsed_data_full = json.loads(llm_parsed)
                    parsed_data_core = parsed_data_full.get("db_friendly", {})
                    if not isinstance(parsed_data_full, dict):
                        raise TypeError("Missing job_data data structure")
                    if not parsed_data_core:
                        await message.reply("Could not parse the job description")
                        return

                    feedback_data = await loop.run_in_executor(None, llm.get_feedback, parsed_data_core)
                    if not feedback_data:
                        await message.reply("feedback is empty")
                        return

                    await loop.run_in_executor(None, db_process.add_job_to_db, parsed_data_full, feedback_data, message.id, db_config)

                    await _post_to_channel(channel_name="processed-jobs", parsed_data=parsed_data_core, feedback_data=feedback_data, message=message)
                except Exception as e:
                    await message.reply(f"ERROR: {str(e)}\n")
                    logger.error(f"Error processing attachment: {e}")
                    return None

    except discord.Forbidden as e:
        logger.error(f"Missing permissions: {e}")
        await message.reply(f"Permission denied! Error: {str(e)}")
    except discord.HTTPException as e:
        logger.error(f"HTTP error: {e}")
        await message.reply(f"HTTP error: {str(e)}")
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        await message.reply(f"Unexpected error: {str(e)}")
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

    # register event handler - on_message -> process_discord_job
    async def on_message(self, message: discord.Message) -> None:
        logger.info("NEW message received")
        await process_discord_job(message, self.db_config)

    # register event handler - on_ready -> process_discord_job for last 30 messages in each channel
    async def on_ready(self):
        logger.info(f'Bot is ready and connected to {len(self.guilds)} guild(s)')
        for guild in self.guilds:
            for channel in guild.text_channels:
                if channel.permissions_for(guild.me).send_messages:
                    async for message in channel.history(limit=30):
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
