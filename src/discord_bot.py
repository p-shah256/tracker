import json
from pathlib import Path
import asyncio
import discord
from discord.ext import commands
from yaml import safe_load
import tempfile
import os
from dotenv import load_dotenv
from llm import parse_job_desc, get_tailored, logger

load_dotenv()

intents = discord.Intents.default()
intents.message_content = True
bot = commands.Bot(command_prefix="!", intents=intents)

@bot.event
async def on_ready():
    logger.info(f"Bot is ready as {bot.user}")

@bot.event
async def on_message(message):
    if message.author == bot.user or bot.user is None:
        return
    if bot.user.mentioned_in(message) and any( att.filename.endswith(".html") for att in message.attachments):
        await process_job_posting(message)
    await bot.process_commands(message)


async def process_job_posting(message):
    await message.add_reaction("⏳")

    try:
        html_attachment = next(
            att for att in message.attachments if att.filename.endswith(".html")
        )
        with tempfile.NamedTemporaryFile(suffix=".html", delete=False) as temp_file:
            file_path = Path(temp_file.name)
            await html_attachment.save(file_path)

        job_data = await asyncio.to_thread(parse_job_desc, file_path)
        if job_data is None:
            return

        resume_path = Path("./Pranchal_Shah_CV.yaml")
        if not resume_path.exists():
            raise FileNotFoundError(f"Resume file not found at {resume_path}")

        with open(resume_path, "r") as file:
            resume = safe_load(file)
            cv = resume.get("cv", {})
            sections = cv.get("sections", {})
            relevant_parts = {
                "technical_skills": sections.get("technical_skills", []),
                "professional_experience": sections.get("professional_experience", []),
                "projects": sections.get("projects", []),
            }
            if not relevant_parts["technical_skills"] or not relevant_parts["professional_experience"] or not relevant_parts["projects"]:
                raise ValueError("Resume sections not found")
            tailored_data = await asyncio.to_thread(get_tailored, job_data, relevant_parts)
            logger.debug(f"Tailored data: {tailored_data}")

            await message.add_reaction("✅")

    except Exception as e:
        logger.error(f"Error processing job: {e}", exc_info=True)
        await message.clear_reactions()
        await message.add_reaction("❌")
        await message.reply(f"Error: {str(e)}")

bot.run(os.getenv("DISCORD_BOT_TOKEN"))
