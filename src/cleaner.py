import json
from bs4 import BeautifulSoup
import re

def clean_html(html_content):
   soup = BeautifulSoup(html_content, 'html.parser')
   
   # Remove non-content and hidden elements
   for tag in soup.find_all(['script', 'style', 'nav', 'header', 'footer', 'iframe', 'img']):
       tag.decompose()
   
   for element in soup.find_all(attrs={'style': re.compile(r'display:\s*none|visibility:\s*hidden')}):
       element.decompose()
   
   # Remove navigation/UI elements
   nav_pattern = re.compile(r'(menu|nav|footer|header|sidebar|banner|cookie|popup)', re.I)
   for element in soup.find_all(class_=nav_pattern):
       element.decompose()
   
   # Extract text
   text = ' '.join(soup.stripped_strings)
   return re.sub(r'\s+', ' ', text).strip()

def clean_llm_response(text: str) -> str:
    """Clean LLM response and return valid JSON string"""

    if not text.strip():
        raise ValueError("Empty response received")

    if '```json' in text:
        text = text.split('```json')[1].split('```')[0].strip()
    elif '```' in text:
        text = text.split('```')[1].split('```')[0].strip()

    try:
        return json.dumps(json.loads(text), indent=2)
    except json.JSONDecodeError:
        text = re.sub(r'[*_`#]', '', text)  # Remove markdown
        try:
            return json.dumps(json.loads(text.strip()), indent=2)
        except json.JSONDecodeError as e:
            raise ValueError(f"Invalid JSON: {text[:200]}...")
